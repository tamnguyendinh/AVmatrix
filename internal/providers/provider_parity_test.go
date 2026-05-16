package providers_test

import (
	"context"
	"testing"
	"time"

	"github.com/tamnguyendinh/avmatrix-go/internal/parser"
	cppprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/cpp"
	csharpprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/csharp"
	dartprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/dart"
	goprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/golang"
	javaprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/java"
	kotlinprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/kotlin"
	phpprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/php"
	pythonprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/python"
	rubyprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/ruby"
	rustprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/rust"
	swiftprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/swift"
	tsjsprovider "github.com/tamnguyendinh/avmatrix-go/internal/providers/tsjs"
	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
	"github.com/tamnguyendinh/avmatrix-go/internal/scopeir"
)

func TestProviderCallExtractionParityCoversFormsReceiversAndArity(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		lang  scanner.Language
		input string
		want  []callExpectation
	}{
		{
			name: "typescript",
			path: "src/app.ts",
			lang: scanner.TypeScript,
			input: `class User { save(value: string) {} }
function free(left: number, right: number) { return left + right; }
function run(user?: User) {
  free(1, 2);
  user?.save("x");
  this?.save("y");
  new User();
}
`,
			want: []callExpectation{
				{name: "free", form: scopeir.CallFree, arity: intPtr(2)},
				{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
				{name: "save", form: scopeir.CallMember, receiver: "this", arity: intPtr(1)},
				{name: "User", form: scopeir.CallConstructor, arity: intPtr(0)},
			},
		},
		{
			name: "javascript",
			path: "src/app.js",
			lang: scanner.JavaScript,
			input: `class User { save(value) {} }
function free(value) { return value; }
function run(user) {
  free(1);
  user.save("x");
}
`,
			want: []callExpectation{
				{name: "free", form: scopeir.CallFree, arity: intPtr(1)},
				{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
			},
		},
		{
			name: "python",
			path: "src/app.py",
			lang: scanner.Python,
			input: `class User:
    def save(self, value):
        pass

def free(value):
    return value

def run(user):
    free(1)
    user.save("x")
    User()
`,
			want: []callExpectation{
				{name: "free", form: scopeir.CallFree, arity: intPtr(1)},
				{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
				{name: "User", form: scopeir.CallConstructor},
			},
		},
		{
			name: "java",
			path: "src/App.java",
			lang: scanner.Java,
			input: `class User { void save(String value) {} }
class App {
  void helper() {}
  void run(User user) {
    helper();
    user.save("x");
    User.staticSave();
    new User();
  }
}
`,
			want: []callExpectation{
				{name: "helper", form: scopeir.CallMember, arity: intPtr(0)},
				{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
				{name: "staticSave", form: scopeir.CallMember, receiver: "User", arity: intPtr(0)},
				{name: "User", form: scopeir.CallConstructor, arity: intPtr(0)},
			},
		},
		{
			name: "csharp",
			path: "src/App.cs",
			lang: scanner.CSharp,
			input: `class User { public void Save(string value) {} }
class App {
  void Helper() {}
  void Run(User user) {
    Helper();
    user.Save("x");
    User.StaticSave();
    new User();
  }
}
`,
			want: []callExpectation{
				{name: "Helper", form: scopeir.CallFree, arity: intPtr(0)},
				{name: "Save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
				{name: "StaticSave", form: scopeir.CallMember, receiver: "User", arity: intPtr(0)},
			},
		},
		{
			name: "go",
			path: "src/app.go",
			lang: scanner.Go,
			input: `package app

type User struct{}
func (u User) Save(value string) {}
func free(value int) int { return value }
func run(user User) {
	free(1)
	user.Save("x")
	User{}
}
`,
			want: []callExpectation{
				{name: "free", form: scopeir.CallFree, arity: intPtr(1)},
				{name: "Save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
				{name: "User", form: scopeir.CallConstructor, arity: intPtr(0)},
			},
		},
		{
			name: "rust",
			path: "src/app.rs",
			lang: scanner.Rust,
			input: `struct User {}
impl User { fn save(&self, value: &str) {} }
fn free(value: i32) -> i32 { value }
fn run(user: User) {
    free(1);
    user.save("x");
    User {};
}
`,
			want: []callExpectation{
				{name: "free", form: scopeir.CallFree, arity: intPtr(1)},
				{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
				{name: "User", form: scopeir.CallConstructor},
			},
		},
		{
			name: "cpp",
			path: "src/app.cpp",
			lang: scanner.CPlusPlus,
			input: `class User {
public:
  void save(const char* value) {}
};
int free(int value) { return value; }
void run(User user) {
  free(1);
  user.save("x");
}
`,
			want: []callExpectation{
				{name: "free", form: scopeir.CallFree, arity: intPtr(1)},
				{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
			},
		},
		{
			name: "php",
			path: "src/app.php",
			lang: scanner.PHP,
			input: `<?php
class User { public function save($value) {} }
function free($value) { return $value; }
function run($user) {
  free(1);
  $user->save("x");
  new User();
}
`,
			want: []callExpectation{
				{name: "free", form: scopeir.CallFree, arity: intPtr(1)},
				{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
				{name: "User", form: scopeir.CallConstructor},
			},
		},
		{
			name: "ruby",
			path: "src/app.rb",
			lang: scanner.Ruby,
			input: `class User
  def save(value)
  end
end

def free(value)
  value
end

def run(user)
  free(1)
  user.save("x")
end
`,
			want: []callExpectation{
				{name: "free", form: scopeir.CallFree, arity: intPtr(1)},
				{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
			},
		},
		{
			name: "kotlin",
			path: "src/App.kt",
			lang: scanner.Kotlin,
			input: `class User { fun save(value: String) {} }
fun free(value: Int) = value
fun run(user: User) {
  free(1)
  user.save("x")
  User()
}
`,
			want: []callExpectation{
				{name: "free", form: scopeir.CallMember, arity: intPtr(1)},
				{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(1)},
				{name: "User", form: scopeir.CallConstructor, arity: intPtr(0)},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ir := extractScopeIR(t, tt.path, "hash-"+tt.name, tt.lang, tt.input)
			for _, want := range tt.want {
				requireCall(t, ir, want)
			}
		})
	}
}

func TestProviderCallExtractionLegacyHelperEdges(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		lang  scanner.Language
		input string
		want  []callExpectation
	}{
		{
			name: "python self receiver",
			path: "src/app.py",
			lang: scanner.Python,
			input: `class User:
    def save(self):
        pass
    def run(self):
        self.save()
`,
			want: []callExpectation{{name: "save", form: scopeir.CallMember, receiver: "self", arity: intPtr(0)}},
		},
		{
			name: "go selector receiver",
			path: "src/app.go",
			lang: scanner.Go,
			input: `package app

type User struct{}
func (u User) Save() {}
func run(user User) {
	user.Save()
}
`,
			want: []callExpectation{{name: "Save", form: scopeir.CallMember, receiver: "user", arity: intPtr(0)}},
		},
		{
			name: "rust scoped call remains free",
			path: "src/app.rs",
			lang: scanner.Rust,
			input: `struct User;
impl User {
    fn new() -> User { User }
}
fn run() {
    User::new();
}
`,
			want: []callExpectation{{name: "new", form: scopeir.CallFree, arity: intPtr(0)}},
		},
		{
			name: "cpp field expression receiver",
			path: "src/app.cpp",
			lang: scanner.CPlusPlus,
			input: `class User {
public:
  void save() {}
};
void run(User user) {
  user.save();
}
`,
			want: []callExpectation{{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(0)}},
		},
		{
			name: "kotlin safe navigation receiver",
			path: "src/App.kt",
			lang: scanner.Kotlin,
			input: `class User { fun save() {} }
fun run(user: User?) {
  user?.save()
}
`,
			want: []callExpectation{{name: "save", form: scopeir.CallMember, receiver: "user", arity: intPtr(0)}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ir := extractScopeIR(t, tt.path, "hash-"+tt.name, tt.lang, tt.input)
			for _, want := range tt.want {
				requireCall(t, ir, want)
			}
		})
	}
}

func TestProviderOwnerExtractionParityCoversLegacyHasMethodCases(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		lang     scanner.Language
		input    string
		owners   []ownerExpectation
		noOwners []definitionExpectation
	}{
		{
			name: "typescript class interface and top level function",
			path: "src/owners.ts",
			lang: scanner.TypeScript,
			input: `interface Runnable {
  run(): void;
}
class Service {
  save(): void {}
}
function free(): void {}
`,
			owners: []ownerExpectation{
				{ownerName: "Runnable", ownerLabel: scopeir.NodeInterface, memberQualifiedName: "Runnable.run", memberLabel: scopeir.NodeMethod},
				{ownerName: "Service", ownerLabel: scopeir.NodeClass, memberQualifiedName: "Service.save", memberLabel: scopeir.NodeMethod},
			},
			noOwners: []definitionExpectation{{name: "free", label: scopeir.NodeFunction}},
		},
		{
			name: "python class method and top level function",
			path: "src/owners.py",
			lang: scanner.Python,
			input: `class Service:
    def save(self):
        pass

def free():
    pass
`,
			owners: []ownerExpectation{
				{ownerName: "Service", ownerLabel: scopeir.NodeClass, memberQualifiedName: "Service.save", memberLabel: scopeir.NodeMethod},
			},
			noOwners: []definitionExpectation{{name: "free", label: scopeir.NodeFunction}},
		},
		{
			name: "java class and interface methods",
			path: "src/Owners.java",
			lang: scanner.Java,
			input: `interface Runnable { void run(); }
class Service {
  void save() {}
}
`,
			owners: []ownerExpectation{
				{ownerName: "Runnable", ownerLabel: scopeir.NodeInterface, memberQualifiedName: "Runnable.run", memberLabel: scopeir.NodeMethod},
				{ownerName: "Service", ownerLabel: scopeir.NodeClass, memberQualifiedName: "Service.save", memberLabel: scopeir.NodeMethod},
			},
		},
		{
			name: "csharp class and interface methods",
			path: "src/Owners.cs",
			lang: scanner.CSharp,
			input: `interface IRunnable { void Run(); }
class Service {
  void Save() {}
}
`,
			owners: []ownerExpectation{
				{ownerName: "IRunnable", ownerLabel: scopeir.NodeInterface, memberQualifiedName: "IRunnable.Run", memberLabel: scopeir.NodeMethod},
				{ownerName: "Service", ownerLabel: scopeir.NodeClass, memberQualifiedName: "Service.Save", memberLabel: scopeir.NodeMethod},
			},
		},
		{
			name: "go receiver method and top level function",
			path: "src/owners.go",
			lang: scanner.Go,
			input: `package owners

type Service struct{}
func (s Service) Save() {}
func Free() {}
`,
			owners: []ownerExpectation{
				{ownerName: "Service", ownerLabel: scopeir.NodeStruct, memberQualifiedName: "Service.Save", memberLabel: scopeir.NodeMethod},
			},
			noOwners: []definitionExpectation{{name: "Free", label: scopeir.NodeFunction}},
		},
		{
			name: "rust trait impl and free function",
			path: "src/owners.rs",
			lang: scanner.Rust,
			input: `trait Runnable {
    fn run(&self);
}
struct Service;
impl Service {
    fn save(&self) {}
}
impl Runnable for Service {
    fn run(&self) {}
}
fn free() {}
`,
			owners: []ownerExpectation{
				{ownerName: "Runnable", ownerLabel: scopeir.NodeTrait, memberQualifiedName: "Runnable.run", memberLabel: scopeir.NodeMethod},
				{ownerName: "Service", ownerLabel: scopeir.NodeStruct, memberQualifiedName: "Service.save", memberLabel: scopeir.NodeMethod},
				{ownerName: "Service", ownerLabel: scopeir.NodeStruct, memberQualifiedName: "Service.run", memberLabel: scopeir.NodeMethod},
			},
			noOwners: []definitionExpectation{{name: "free", label: scopeir.NodeFunction}},
		},
		{
			name: "cpp class method and free function",
			path: "src/owners.cpp",
			lang: scanner.CPlusPlus,
			input: `class Service {
public:
  void save() {}
};
void free() {}
`,
			owners: []ownerExpectation{
				{ownerName: "Service", ownerLabel: scopeir.NodeClass, memberQualifiedName: "Service.save", memberLabel: scopeir.NodeMethod},
			},
			noOwners: []definitionExpectation{{name: "free", label: scopeir.NodeFunction}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ir := extractScopeIR(t, tt.path, "hash-"+tt.name, tt.lang, tt.input)
			for _, want := range tt.owners {
				requireOwnedDefinition(t, ir, want)
			}
			for _, want := range tt.noOwners {
				requireDefinitionWithoutOwner(t, ir, want)
			}
		})
	}
}

func TestProviderHeritageExtractionParityCoversLanguageSpecificForms(t *testing.T) {
	tsIR := extractScopeIR(t, "src/app.ts", "hash-ts-heritage", scanner.TypeScript, `interface Named {}
class Base {}
class User extends Base implements Named {}
`)
	requireHeritage(t, tsIR, heritageExpectation{name: "Base", kind: scopeir.HeritageExtends})
	requireHeritage(t, tsIR, heritageExpectation{name: "Named", kind: scopeir.HeritageImplements})

	goIR := extractScopeIR(t, "src/app.go", "hash-go-heritage", scanner.Go, `package app

type Animal struct{}
type Dog struct {
	Animal
	Breed string
}
`)
	requireHeritage(t, goIR, heritageExpectation{name: "Animal", kind: scopeir.HeritageExtends})
	requireNoHeritage(t, goIR, heritageExpectation{name: "string", kind: scopeir.HeritageExtends})
	requireNoHeritage(t, goIR, heritageExpectation{name: "Breed", kind: scopeir.HeritageExtends})

	rubyIR := extractScopeIR(t, "src/app.rb", "hash-ruby-heritage", scanner.Ruby, `module Serializable
end
module ClassMethods
end
module Instrumented
end
class User
  include Serializable
  extend ClassMethods
  prepend Instrumented
end
`)
	requireHeritage(t, rubyIR, heritageExpectation{name: "Serializable", kind: scopeir.HeritageInclude})
	requireHeritage(t, rubyIR, heritageExpectation{name: "ClassMethods", kind: scopeir.HeritageExtend})
	requireHeritage(t, rubyIR, heritageExpectation{name: "Instrumented", kind: scopeir.HeritagePrepend})

	javaIR := extractScopeIR(t, "src/User.java", "hash-java-heritage", scanner.Java, `interface Named {}
class Base {}
class User extends Base implements Named {}
`)
	requireHeritage(t, javaIR, heritageExpectation{name: "Base", kind: scopeir.HeritageExtends})
	requireHeritage(t, javaIR, heritageExpectation{name: "Named", kind: scopeir.HeritageImplements})

	csharpIR := extractScopeIR(t, "src/User.cs", "hash-csharp-heritage", scanner.CSharp, `interface INamed {}
class Base {}
class User : Base, INamed {}
`)
	requireHeritage(t, csharpIR, heritageExpectation{name: "Base", kind: scopeir.HeritageExtends})
	requireHeritage(t, csharpIR, heritageExpectation{name: "INamed", kind: scopeir.HeritageImplements})

	rustIR := extractScopeIR(t, "src/user.rs", "hash-rust-heritage", scanner.Rust, `trait Named {}
struct User;
impl Named for User {}
`)
	requireHeritage(t, rustIR, heritageExpectation{name: "Named", kind: scopeir.HeritageTraitImpl})

	phpIR := extractScopeIR(t, "src/User.php", "hash-php-heritage", scanner.PHP, `<?php
interface Named {}
class Base {}
class User extends Base implements Named {}
`)
	requireHeritage(t, phpIR, heritageExpectation{name: "Base", kind: scopeir.HeritageExtends})
	requireHeritage(t, phpIR, heritageExpectation{name: "Named", kind: scopeir.HeritageImplements})

	dartIR := extractScopeIR(t, "lib/user.dart", "hash-dart-heritage", scanner.Dart, `class Base {}
class User extends Base {}
`)
	requireHeritage(t, dartIR, heritageExpectation{name: "Base", kind: scopeir.HeritageExtends})
}

func TestProviderQualifiedNamesCoverNamespacesPackagesModulesAndTopLevelClasses(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		lang  scanner.Language
		input string
		want  string
	}{
		{
			name:  "csharp namespace",
			path:  "src/Services/User.cs",
			lang:  scanner.CSharp,
			input: "namespace Services.Auth;\npublic class User {}\n",
			want:  "Services.Auth.User",
		},
		{
			name:  "java package",
			path:  "src/models/Config.java",
			lang:  scanner.Java,
			input: "package com.example.models;\nclass Config {}\n",
			want:  "com.example.models.Config",
		},
		{
			name:  "ruby module",
			path:  "lib/admin/user.rb",
			lang:  scanner.Ruby,
			input: "module Admin\n  class User\n  end\nend\n",
			want:  "Admin.User",
		},
		{
			name:  "typescript top level",
			path:  "src/plain-user.ts",
			lang:  scanner.TypeScript,
			input: "export class User {}\n",
			want:  "User",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ir := extractScopeIR(t, test.path, "hash-"+test.name, test.lang, test.input)
			requireDefinitionQualifiedName(t, ir, scopeir.NodeClass, test.want)
		})
	}
}

func TestProviderImportExtractionParityCoversLanguageSpecificForms(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		lang  scanner.Language
		input string
		want  []importExpectation
	}{
		{
			name: "typescript default alias reexport",
			path: "src/app.ts",
			lang: scanner.TypeScript,
			input: `import User, { format as fmt } from './user';
export { Service as UserService } from './service';
`,
			want: []importExpectation{
				{kind: scopeir.ImportNamed, local: "User", imported: "default", target: "./user"},
				{kind: scopeir.ImportAlias, local: "fmt", imported: "format", alias: "fmt", target: "./user"},
				{kind: scopeir.ImportReexport, local: "UserService", imported: "Service", alias: "UserService", target: "./service"},
			},
		},
		{
			name: "javascript named import",
			path: "src/app.js",
			lang: scanner.JavaScript,
			input: `import { createService } from './factory';
export function start() { createService(); }
`,
			want: []importExpectation{{kind: scopeir.ImportNamed, local: "createService", imported: "createService", target: "./factory"}},
		},
		{
			name: "csharp alias and namespace using",
			path: "src/App.cs",
			lang: scanner.CSharp,
			input: `using System;
using RepoAlias = App.Models.Repository;
class App {}
`,
			want: []importExpectation{
				{kind: scopeir.ImportNamed, local: "System", imported: "System", target: "System"},
				{kind: scopeir.ImportAlias, local: "RepoAlias", imported: "Repository", alias: "RepoAlias", target: "App.Models.Repository"},
			},
		},
		{
			name: "go stdlib and package alias",
			path: "cmd/app/main.go",
			lang: scanner.Go,
			input: `package main

import (
	"context"
	repoPkg "example.com/app/repo"
)

func main() { _ = context.TODO; _ = repoPkg.Repo{} }
`,
			want: []importExpectation{
				{kind: scopeir.ImportNamed, local: "context", imported: "context", target: "context"},
				{kind: scopeir.ImportAlias, local: "repoPkg", imported: "repo", alias: "repoPkg", target: "example.com/app/repo"},
			},
		},
		{
			name: "java wildcard and member import",
			path: "src/app/Main.java",
			lang: scanner.Java,
			input: `package app;
import com.example.models.*;
import com.example.util.FormatUtil.format;
class Main {}
`,
			want: []importExpectation{
				{kind: scopeir.ImportWildcard, local: "", imported: "*", target: "com.example.models.*"},
				{kind: scopeir.ImportNamed, local: "format", imported: "format", target: "com.example.util.FormatUtil.format"},
			},
		},
		{
			name: "kotlin alias and wildcard import",
			path: "src/App.kt",
			lang: scanner.Kotlin,
			input: `package app
import com.example.models.User as AppUser
import com.example.util.*
fun main() {}
`,
			want: []importExpectation{
				{kind: scopeir.ImportNamed, local: "User", imported: "User", target: "com.example.models.User"},
				{kind: scopeir.ImportNamed, local: "util", imported: "util", target: "com.example.util"},
			},
		},
		{
			name: "python alias and from import",
			path: "src/app.py",
			lang: scanner.Python,
			input: `import os.path as osp
from app.models import User as AppUser
`,
			want: []importExpectation{
				{kind: scopeir.ImportAlias, local: "osp", imported: "path", alias: "osp", target: "os.path"},
				{kind: scopeir.ImportAlias, local: "AppUser", imported: "User", alias: "AppUser", target: "app.models.User"},
			},
		},
		{
			name: "ruby require forms",
			path: "lib/app.rb",
			lang: scanner.Ruby,
			input: `require "json"
require_relative "./models/user"
`,
			want: []importExpectation{
				{kind: scopeir.ImportNamed, local: "json", imported: "json", target: "json"},
				{kind: scopeir.ImportNamed, local: "user", imported: "user", target: "./models/user"},
			},
		},
		{
			name: "rust grouped imports",
			path: "src/main.rs",
			lang: scanner.Rust,
			input: `use crate::models::{User, Repo as Store};
fn main() {}
`,
			want: []importExpectation{{kind: scopeir.ImportNamed, local: "Store", imported: "Store", target: "crate::models::{User, Repo as Store}"}},
		},
		{
			name: "cpp includes",
			path: "src/app.cpp",
			lang: scanner.CPlusPlus,
			input: `#include "models/user.hpp"
#include <vector>
int main() { return 0; }
`,
			want: []importExpectation{
				{kind: scopeir.ImportNamed, local: "models/user.hpp", imported: "models/user.hpp", target: "models/user.hpp"},
				{kind: scopeir.ImportNamed, local: "vector", imported: "vector", target: "vector"},
			},
		},
		{
			name: "dart sdk package relative and export",
			path: "lib/main.dart",
			lang: scanner.Dart,
			input: `import 'dart:async';
import 'package:my_app/models/user.dart';
import './repo.dart' as repo;
export 'src/public.dart';
class App {}
`,
			want: []importExpectation{
				{kind: scopeir.ImportNamed, local: "dart:async", imported: "dart:async", target: "dart:async"},
				{kind: scopeir.ImportNamed, local: "user", imported: "user", target: "package:my_app/models/user.dart"},
				{kind: scopeir.ImportAlias, local: "repo", imported: "repo", alias: "repo", target: "./repo.dart"},
				{kind: scopeir.ImportNamed, local: "public", imported: "public", target: "src/public.dart"},
			},
		},
		{
			name: "swift framework import",
			path: "Sources/App/App.swift",
			lang: scanner.Swift,
			input: `import Foundation
class App {}
`,
			want: []importExpectation{{kind: scopeir.ImportNamed, local: "Foundation", imported: "Foundation", target: "Foundation"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ir := extractScopeIR(t, tt.path, "hash-"+tt.name, tt.lang, tt.input)
			for _, want := range tt.want {
				requireImport(t, ir, want)
			}
		})
	}
}

type callExpectation struct {
	name     string
	form     scopeir.CallForm
	receiver string
	arity    *int
}

type ownerExpectation struct {
	ownerName           string
	ownerLabel          scopeir.NodeLabel
	memberQualifiedName string
	memberLabel         scopeir.NodeLabel
}

type definitionExpectation struct {
	name  string
	label scopeir.NodeLabel
}

type heritageExpectation struct {
	name string
	kind scopeir.HeritageKind
}

type importExpectation struct {
	kind     scopeir.ImportKind
	local    string
	imported string
	alias    string
	target   string
}

func extractScopeIR(t *testing.T, filePath string, fileHash string, language scanner.Language, source string) scopeir.ScopeIR {
	t.Helper()
	pool := parser.NewPool(nil, parser.PoolOptions{ParseTimeout: time.Second})
	defer pool.Close()

	parsed, err := pool.Parse(context.Background(), parser.Request{
		FilePath: filePath,
		Language: language,
		Source:   []byte(source),
	})
	if err != nil {
		t.Fatalf("parse %s failed: %v", filePath, err)
	}
	defer parsed.Close()

	switch language {
	case scanner.TypeScript, scanner.JavaScript:
		ir, err := tsjsprovider.Extract(tsjsprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.Python:
		ir, err := pythonprovider.Extract(pythonprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.Java:
		ir, err := javaprovider.Extract(javaprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.CSharp:
		ir, err := csharpprovider.Extract(csharpprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.Go:
		ir, err := goprovider.Extract(goprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.Rust:
		ir, err := rustprovider.Extract(rustprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.CPlusPlus:
		ir, err := cppprovider.Extract(cppprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.PHP:
		ir, err := phpprovider.Extract(phpprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.Ruby:
		ir, err := rubyprovider.Extract(rubyprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.Kotlin:
		ir, err := kotlinprovider.Extract(kotlinprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.Dart:
		ir, err := dartprovider.Extract(dartprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	case scanner.Swift:
		ir, err := swiftprovider.Extract(swiftprovider.Request{
			FilePath: filePath,
			FileHash: fileHash,
			Language: language,
			Source:   []byte(source),
			Root:     parsed.Tree.RootNode(),
		})
		requireNoExtractError(t, err)
		return ir
	default:
		t.Fatalf("unsupported test language %q", language)
	}
	return scopeir.ScopeIR{}
}

func requireDefinitionQualifiedName(t *testing.T, ir scopeir.ScopeIR, label scopeir.NodeLabel, qualifiedName string) {
	t.Helper()
	for _, def := range ir.Definitions {
		if def.Label == label && def.QualifiedName == qualifiedName {
			return
		}
	}
	t.Fatalf("missing %s qualifiedName %q in %#v", label, qualifiedName, ir.Definitions)
}

func requireOwnedDefinition(t *testing.T, ir scopeir.ScopeIR, want ownerExpectation) {
	t.Helper()
	owner := findDefinition(t, ir, definitionExpectation{name: want.ownerName, label: want.ownerLabel})
	member := findQualifiedDefinition(t, ir, want.memberQualifiedName, want.memberLabel)
	if member.OwnerID != owner.ID {
		t.Fatalf("%s owner = %q, want %q from %#v", want.memberQualifiedName, member.OwnerID, owner.ID, member)
	}
}

func requireDefinitionWithoutOwner(t *testing.T, ir scopeir.ScopeIR, want definitionExpectation) {
	t.Helper()
	def := findDefinition(t, ir, want)
	if def.OwnerID != "" {
		t.Fatalf("%s/%s owner = %q, want empty", want.label, want.name, def.OwnerID)
	}
}

func findDefinition(t *testing.T, ir scopeir.ScopeIR, want definitionExpectation) scopeir.DefinitionFact {
	t.Helper()
	for _, def := range ir.Definitions {
		if def.Name == want.name && def.Label == want.label {
			return def
		}
	}
	t.Fatalf("missing %s/%s in %#v", want.label, want.name, ir.Definitions)
	return scopeir.DefinitionFact{}
}

func findQualifiedDefinition(t *testing.T, ir scopeir.ScopeIR, qualified string, label scopeir.NodeLabel) scopeir.DefinitionFact {
	t.Helper()
	for _, def := range ir.Definitions {
		if def.QualifiedName == qualified && def.Label == label {
			return def
		}
	}
	t.Fatalf("missing %s qualifiedName %q in %#v", label, qualified, ir.Definitions)
	return scopeir.DefinitionFact{}
}

func requireCall(t *testing.T, ir scopeir.ScopeIR, want callExpectation) {
	t.Helper()
	for _, call := range ir.Calls {
		if call.Name != want.name || call.CallForm != want.form || call.ExplicitReceiver != want.receiver {
			continue
		}
		if want.arity == nil || optionalIntEqual(call.Arity, want.arity) {
			return
		}
	}
	t.Fatalf("missing call name=%s form=%s receiver=%q arity=%s in %#v", want.name, want.form, want.receiver, optionalIntString(want.arity), ir.Calls)
}

func requireImport(t *testing.T, ir scopeir.ScopeIR, want importExpectation) {
	t.Helper()
	for _, item := range ir.Imports {
		if item.Kind == want.kind && item.LocalName == want.local && item.ImportedName == want.imported && item.Alias == want.alias && item.TargetRaw != nil && *item.TargetRaw == want.target {
			return
		}
	}
	t.Fatalf("missing import kind=%s local=%s imported=%s alias=%s target=%s in %#v", want.kind, want.local, want.imported, want.alias, want.target, ir.Imports)
}

func requireHeritage(t *testing.T, ir scopeir.ScopeIR, want heritageExpectation) {
	t.Helper()
	for _, item := range ir.Heritage {
		if item.Name == want.name && item.Kind == want.kind {
			return
		}
	}
	t.Fatalf("missing heritage name=%s kind=%s in %#v", want.name, want.kind, ir.Heritage)
}

func requireNoHeritage(t *testing.T, ir scopeir.ScopeIR, want heritageExpectation) {
	t.Helper()
	for _, item := range ir.Heritage {
		if item.Name == want.name && item.Kind == want.kind {
			t.Fatalf("unexpected heritage name=%s kind=%s in %#v", want.name, want.kind, ir.Heritage)
		}
	}
}

func requireNoExtractError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("extract failed: %v", err)
	}
}

func optionalIntEqual(left *int, right *int) bool {
	if left == nil || right == nil {
		return left == right
	}
	return *left == *right
}

func optionalIntString(value *int) string {
	if value == nil {
		return "<nil>"
	}
	switch *value {
	case 0:
		return "0"
	case 1:
		return "1"
	case 2:
		return "2"
	default:
		return "n"
	}
}

func intPtr(value int) *int {
	return &value
}
