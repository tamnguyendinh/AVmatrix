package sfc

import (
	"strings"
	"testing"

	"github.com/tamnguyendinh/avmatrix-go/internal/scanner"
)

func TestExtractHTMLScriptPrefersFirstInlineScriptAndPreservesLineOffset(t *testing.T) {
	source := []byte(`<template>
  <div>Hello</div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
const count = ref(0);
</script>
`)

	block := ExtractHTMLScript(source)
	if block.Language != scanner.TypeScript {
		t.Fatalf("Language = %q, want TypeScript", block.Language)
	}
	if !strings.Contains(block.Source, "const count = ref(0);") {
		t.Fatalf("script source missing setup content: %q", block.Source)
	}
	if got := leadingNewlines(block.Source); got != 5 {
		t.Fatalf("leading newlines = %d, want 5", got)
	}
}

func TestExtractHTMLScriptHandlesJavaScriptAndMultilineAttributes(t *testing.T) {
	source := []byte(`<template><div /></template>
<script
  setup
  lang='ts'
>
const value: number = 1;
</script>
`)

	block := ExtractHTMLScript(source)
	if block.Language != scanner.TypeScript {
		t.Fatalf("Language = %q, want TypeScript", block.Language)
	}
	if !strings.Contains(block.Source, "const value: number = 1;") {
		t.Fatalf("script source missing multiline-tag content: %q", block.Source)
	}

	js := ExtractHTMLScript([]byte(`<script>export default { name: 'Plain' };</script>`))
	if js.Language != scanner.JavaScript || !strings.Contains(js.Source, "Plain") {
		t.Fatalf("plain script block = %#v, want JavaScript content", js)
	}
}

func TestExtractHTMLScriptPrefersSetupBlockWhenBothScriptsExist(t *testing.T) {
	source := []byte(`<script lang="ts">
export default { inheritAttrs: false };
</script>

<script setup lang="ts">
import { ref } from 'vue';
const name = ref('test');
</script>
`)

	block := ExtractHTMLScript(source)
	if block.Language != scanner.TypeScript {
		t.Fatalf("Language = %q, want TypeScript", block.Language)
	}
	if !strings.Contains(block.Source, "const name = ref('test')") || strings.Contains(block.Source, "inheritAttrs") {
		t.Fatalf("script source did not prefer setup block: %q", block.Source)
	}
}

func TestExtractHTMLScriptSkipsExternalScriptsAndReturnsEmptyWhenMissing(t *testing.T) {
	source := []byte(`<script src="./external.ts"></script>
<script type="text/typescript">
export const local = true;
</script>
`)

	block := ExtractHTMLScript(source)
	if block.Language != scanner.TypeScript {
		t.Fatalf("Language = %q, want TypeScript", block.Language)
	}
	if strings.Contains(block.Source, "external") || !strings.Contains(block.Source, "local") {
		t.Fatalf("script source did not skip external script: %q", block.Source)
	}

	empty := ExtractHTMLScript([]byte(`<template><button /></template>`))
	if empty.Language != scanner.TypeScript || empty.Source != "" {
		t.Fatalf("empty script block = %#v, want TypeScript identity with empty source", empty)
	}
}

func TestExtractTemplateComponentsFindsPascalCaseTags(t *testing.T) {
	source := []byte(`<template>
  <div>
    <MyButton @click="doSomething" />
    <AppHeader title="hello" />
    <span>text</span>
    <MyButton />
  </div>
</template>
`)

	components := ExtractTemplateComponents(source)
	want := []string{"MyButton", "AppHeader"}
	if strings.Join(components, ",") != strings.Join(want, ",") {
		t.Fatalf("components = %#v, want %#v", components, want)
	}
}

func TestExtractTemplateComponentsIgnoresLowercaseAndMissingTemplate(t *testing.T) {
	source := []byte(`<template>
  <div>
    <p>text</p>
    <router-view />
    <transition name="fade">
      <MyComponent />
    </transition>
  </div>
</template>
`)

	if components := ExtractTemplateComponents(source); strings.Join(components, ",") != "MyComponent" {
		t.Fatalf("components = %#v, want MyComponent", components)
	}
	if components := ExtractTemplateComponents([]byte(`<script setup>const x = 1;</script>`)); len(components) != 0 {
		t.Fatalf("components without template = %#v, want empty", components)
	}
}

func leadingNewlines(value string) int {
	count := 0
	for _, r := range value {
		if r != '\n' {
			return count
		}
		count++
	}
	return count
}
