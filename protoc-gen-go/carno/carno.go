// Go support for Protocol Buffers - Google's data interchange format
//
// Copyright 2015 The Go Authors.  All rights reserved.
// https://github.com/golang/protobuf
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of Google Inc. nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

// Package carno outputs carno service descriptions in Go code.
// It runs as a plugin for the Go protocol buffer compiler plugin.
// It is linked in to protoc-gen-go.
package carno

import (
	"fmt"
	"strconv"
	"strings"

	pb "github.com/golang/protobuf/protoc-gen-go/descriptor"
	"github.com/ccsnake/protobuf/protoc-gen-go/generator"
)

// generatedCodeVersion indicates a version of the generated code.
// It is incremented whenever an incompatibility between the generated code and
// the carno package is introduced; the generated code references
// a constant, carno.SupportPackageIsVersionN (where N is generatedCodeVersion).
const generatedCodeVersion = 4

func init() {
	generator.RegisterPlugin(new(carno))
}

// carno is an implementation of the Go protocol buffer compiler's
// plugin architecture.  It generates bindings for carno support.
type carno struct {
	gen *generator.Generator
}

// Name returns the name of this plugin, "carno".
func (g *carno) Name() string {
	return "carno"
}

// Init initializes the plugin.
func (g *carno) Init(gen *generator.Generator) {
	g.gen = gen
}

// Given a type name defined in a .proto, return its object.
// Also record that we're using it, to guarantee the associated import.
func (g *carno) objectNamed(name string) generator.Object {
	g.gen.RecordTypeUse(name)
	return g.gen.ObjectNamed(name)
}

// Given a type name defined in a .proto, return its name as we will print it.
func (g *carno) typeName(str string) string {
	return g.gen.TypeName(g.objectNamed(str))
}

// P forwards to g.gen.P.
func (g *carno) P(args ...interface{}) { g.gen.P(args...) }

// Generate generates code for the services in the given file.
func (g *carno) Generate(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.P("// Reference imports to suppress errors if they are not otherwise used.")
	g.P()

	// Assert version compatibility.
	g.P("// This is a compile-time assertion to ensure that this generated file")
	g.P("// is compatible with the carno package it is being compiled against.")
	g.P()

	g.P(`const ServerName=`, strconv.Quote(file.GetPackage()))

	for i, service := range file.FileDescriptorProto.Service {
		g.generateService(file, service, i)
	}

	g.generateServerPackage(file.GetPackage(), file.FileDescriptorProto.Service)
}

// GenerateImports generates the import declaration for this file.
func (g *carno) GenerateImports(file *generator.FileDescriptor) {
	if len(file.FileDescriptorProto.Service) == 0 {
		return
	}

	g.P("import (")
	g.P(strconv.Quote("carno/client"))
	g.P(strconv.Quote("carno/server"))
	g.P(strconv.Quote("carno/options"))
	g.P(strconv.Quote("carno/common"))
	g.P(strconv.Quote("context"))
	g.P(")")
	g.P()

}

// reservedClientName records whether a client name is reserved on the client side.
var reservedClientName = map[string]bool{
// TODO: do we need any in carno?
}

func unexport(s string) string { return strings.ToLower(s[:1]) + s[1:] }

// generateService generates all the code for the named service.
func (g *carno) generateService(file *generator.FileDescriptor, service *pb.ServiceDescriptorProto, index int) {
	path := fmt.Sprintf("6,%d", index) // 6 means service.

	origServName := service.GetName()
	fullServName := origServName
	// pkg代表服务名
	if pkg := file.GetPackage(); pkg != "" {
		fullServName = pkg + "@" + fullServName
	}
	servName := generator.CamelCase(origServName)

	g.P()
	g.P("// Client API for ", servName, " service")

	// Client interface.
	g.P("type ", servName, "Client interface {")
	for i, method := range service.Method {
		g.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
		g.P(g.generateClientSignature(servName, method))
	}
	g.P("}")
	g.P()

	// Client structure.
	g.P("type ", unexport(servName), "Client struct {")
	g.P("*client.Client")
	g.P("}")
	g.P()

	// NewClient factory.
	g.P("func New", servName, "Client (opts *options.Options) (", servName, "Client, error) {")
	g.P(`c, err := client.NewClient(opts,`, strconv.Quote(file.GetPackage()), ")")
	g.P("if err != nil {return nil, err}")
	g.P("rv := &", unexport(servName), "Client{Client: c}")

	g.P("return rv, nil")
	g.P("}")
	g.P()

	var methodIndex int
	serviceDescVar := "_" + servName + "_serviceDesc"
	// Client method implementations.
	for _, method := range service.Method {
		descExpr := fmt.Sprintf("&%s.Methods[%d]", serviceDescVar, methodIndex)
		methodIndex++
		g.generateClientMethod(file.GetPackage(), origServName, fullServName, serviceDescVar, method, descExpr)
	}

	g.P("// Server API for ", servName, " service")
	// Server interface.
	serverType := servName + "Server"
	g.P("type ", serverType, " interface {")
	for i, method := range service.Method {
		g.gen.PrintComments(fmt.Sprintf("%s,2,%d", path, i)) // 2 means method in a service.
		g.P(g.generateServerSignature(servName, method))
	}
	g.P("}")
	g.P()

	g.generateServerSetting(file)
	g.P()
	// Server registration.
	g.P("func Register", servName, "Server(s server.Server, srv ", serverType, ") {")
	g.P("s.RegisterService(&", serviceDescVar, `, srv)`)

	g.P("}")
	g.P()

	// Service descriptor.
	g.P("var ", serviceDescVar, " = ", "common.ServiceDescribe {")
	g.P("ServiceName: ", strconv.Quote(origServName), ",")
	g.P("Methods: []", "string{")
	for _, method := range service.Method {
		if method.GetServerStreaming() || method.GetClientStreaming() {
			continue
		}
		g.P(strconv.Quote(method.GetName()), ",")
	}
	g.P("},")
	g.P("}")
	g.P()
}

// generateClientSignature returns the client-side signature for a method.
func (g *carno) generateClientSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}
	reqArg := ", in *" + g.typeName(method.GetInputType())
	if method.GetClientStreaming() {
		reqArg = ""
	}
	respName := "*" + g.typeName(method.GetOutputType())
	if method.GetServerStreaming() || method.GetClientStreaming() {
		respName = servName + "_" + generator.CamelCase(origMethName) + "Client"
	}
	return fmt.Sprintf("%s(ctx context.Context%s, opts ...options.Option) (%s, error)", methName, reqArg, respName)

}

func (g *carno) generateClientMethod(pkgName, servName, fullServName, serviceDescVar string, method *pb.MethodDescriptorProto, descExpr string) {
	// methName := generator.CamelCase(method.GetName())
	// inType := g.typeName(method.GetInputType())
	outType := g.typeName(method.GetOutputType())

	g.P("func (c *", unexport(servName), "Client) ", g.generateClientSignature(servName, method), "{")
	g.P("inBytes, err := proto.Marshal(in)")
	g.P("if err != nil{")
	g.P("return nil, err")
	g.P("}")

	// invoke
	g.P("resp, err := c.Client.Invoke(ctx, ", strconv.Quote(servName), ", ", strconv.Quote(method.GetName()), ", opts, inBytes)")
	g.P("if err!=nil{")
	g.P("return nil,err")
	g.P("}")

	// Unmarshal resp
	g.P("out := new(", outType, ")")
	g.P("if err := proto.Unmarshal(resp, out); err != nil {")
	g.P("return nil, err")
	g.P("}")
	g.P("return out, nil")
	g.P("}")
	g.P()
	return
}

// generateServerSignature returns the server-side signature for a method.
func (g *carno) generateServerSignature(servName string, method *pb.MethodDescriptorProto) string {
	origMethName := method.GetName()
	methName := generator.CamelCase(origMethName)
	if reservedClientName[methName] {
		methName += "_"
	}

	var reqArgs []string
	ret := "error"
	reqArgs = append(reqArgs, "context.Context", "*"+g.typeName(method.GetInputType()))
	ret = "(*" + g.typeName(method.GetOutputType()) + ", error)"

	return methName + "(" + strings.Join(reqArgs, ", ") + ") " + ret
}

func (g *carno) generateServerSetting(file *generator.FileDescriptor) {
	pkg := file.GetPackage()
	if pkg == "" {
		panic("empty package")
	}
}

// type Quanmin_core_demo struct {
// 	DemoClient
// 	UserClient
// }

// func NewQuanmin_core_demo(addr string) (*Quanmin_core_demo, error) {
// 	c, err := carno.NewClient(carno.RegistryAddr(addr), carno.ServerName("quanmin.core.demo"))
// 	if err != nil {
// 		return nil, err
// 	}
// 	return &Quanmin_core_demo{
// 		DemoClient: &demoClient{Client: c},
// 		UserClient: &userClient{Client: c},
// 	}, nil
// }
func (g *carno) generateServerPackage(pkg string, services []*pb.ServiceDescriptorProto) {
	camelCasePkgName := generator.CamelCase(strings.Replace(pkg, ".", "_", -1))
	g.P("type ", camelCasePkgName, " struct{")
	for _, service := range services {
		g.P(generator.CamelCase(service.GetName()), "Client")
	}
	g.P("}")
	g.P("")

	g.P("func New", camelCasePkgName, "(opts *options.Options) (*", camelCasePkgName, ",error){")
	g.P("c, err := client.NewClient(opts,", strconv.Quote(pkg), ")")
	g.P("if err!=nil{")
	g.P("return nil,err")
	g.P("}")

	g.P("return &", camelCasePkgName, "{")
	for _, service := range services {
		g.P(generator.CamelCase(service.GetName()), "Client: &", unexport(service.GetName()), "Client{Client:c},")
	}
	g.P("},nil")
	g.P("}")
	g.P("")
}
