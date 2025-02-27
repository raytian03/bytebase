<template>
  <div ref="editorRef" style="height: 100%; width: 100%"></div>
</template>

<script lang="ts" setup>
import {
  onMounted,
  ref,
  toRef,
  toRaw,
  nextTick,
  onUnmounted,
  watch,
  computed,
} from "vue";
import { useI18n } from "vue-i18n";
import { editor as Editor } from "monaco-editor";

import { useMonaco } from "./useMonaco";

import {
  pushNotification,
  useTabStore,
  useSQLEditorStore,
  useSheetStore,
} from "@/store";
import { SQLDialect } from "../../types";
import { useLineDecorations } from "./lineDecorations";

const props = defineProps({
  value: {
    type: String,
    required: true,
  },
  language: {
    type: String,
    default: "mysql",
  },
});

const emit = defineEmits<{
  (e: "update:value", content: string): void;
  (e: "change", content: string): void;
  (e: "change-selection", content: string): void;
  (
    e: "run-query",
    content: {
      explain: boolean;
      query: string;
    }
  ): void;
  (e: "save", content: string): void;
}>();

const editorRef = ref();
const sqlCode = toRef(props, "value");
const language = toRef(props, "language");

const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();
const sheetStore = useSheetStore();
const { t } = useI18n();

const readonly = computed(() => sheetStore.isReadOnly);

let editorInstance: Editor.IStandaloneCodeEditor;

const {
  monaco,
  setPositionAtEndOfLine,
  formatContent,
  setContent,
  completionItemProvider,
} = await useMonaco(language.value);

const init = async () => {
  const model = monaco.editor.createModel(sqlCode.value, toRaw(language.value));

  editorInstance = monaco.editor.create(editorRef.value, {
    model,
    tabSize: 2,
    insertSpaces: true,
    autoClosingQuotes: "always",
    detectIndentation: false,
    folding: false,
    automaticLayout: true,
    theme: "vs-light",
    minimap: {
      enabled: false,
    },
    wordWrap: "on",
    fixedOverflowWidgets: true,
  });

  // add the run query action in context menu
  editorInstance.addAction({
    id: "RunQuery",
    label: "Run Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.Enter],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: async () => {
      const typedValue = editorInstance.getValue();
      const selectedValue = editorInstance
        .getModel()
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-expect-error
        ?.getValueInRange(editorInstance.getSelection()) as string;

      const query = selectedValue || typedValue;
      emit("run-query", { explain: false, query });
    },
  });

  // add the run query action in context menu
  editorInstance.addAction({
    id: "ExplainQuery",
    label: "Explain Query",
    keybindings: [monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyE],
    contextMenuGroupId: "operation",
    contextMenuOrder: 0,
    run: async () => {
      const typedValue = editorInstance.getValue();
      const selectedValue = editorInstance
        .getModel()
        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
        // @ts-expect-error
        ?.getValueInRange(editorInstance.getSelection()) as string;

      const query = selectedValue || typedValue;
      emit("run-query", { explain: true, query });
    },
  });

  // add format sql action in context menu
  editorInstance.addAction({
    id: "FormatSQL",
    label: "Format SQL",
    keybindings: [
      monaco.KeyMod.Alt | monaco.KeyMod.Shift | monaco.KeyCode.KeyF,
    ],
    contextMenuGroupId: "operation",
    contextMenuOrder: 1,
    run: () => {
      if (readonly.value) {
        pushNotification({
          module: "bytebase",
          style: "INFO",
          title: t("sql-editor.notify.sheet-is-read-only"),
        });
        return;
      }
      formatContent(editorInstance, language.value as SQLDialect);
      nextTick(() => setPositionAtEndOfLine(editorInstance));
    },
  });

  // typed something, change the text
  editorInstance.onDidChangeModelContent(() => {
    const value = editorInstance.getValue();
    // emit("update:value", value);
    emit("change", value);
  });

  // when editor change selection, emit change-selection event with selected text
  editorInstance.onDidChangeCursorSelection((e) => {
    const selectedText = editorInstance
      .getModel()
      ?.getValueInRange(e.selection) as string;
    emit("change-selection", selectedText);
  });

  editorInstance.onDidChangeCursorPosition((e) => {
    const { defineLineDecorations, disposeLineDecorations } =
      useLineDecorations(editorInstance, e.position);
    // clear the old decorations
    disposeLineDecorations();

    // define the new decorations
    nextTick(() => {
      defineLineDecorations();
    });
  });

  editorInstance.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => {
    const value = editorInstance.getValue();
    emit("save", value);
  });

  // set the editor focus when the tab is selected
  if (!readonly.value) {
    editorInstance.focus();

    nextTick(() => setPositionAtEndOfLine(editorInstance));
  }

  watch(
    () => readonly.value,
    (readOnly) => {
      if (editorInstance) {
        editorInstance.updateOptions({ readOnly });
      }
    },
    {
      deep: true,
      immediate: true,
    }
  );
};

onMounted(init);

onUnmounted(() => {
  completionItemProvider.dispose();
  editorInstance.dispose();
});

watch(
  () => sqlEditorStore.shouldSetContent,
  () => {
    if (sqlEditorStore.shouldSetContent) {
      sqlEditorStore.setShouldSetContent(false);
      setContent(editorInstance, tabStore.currentTab.statement);
    }
  }
);

// trigger format code from outside
watch(
  () => sqlEditorStore.shouldFormatContent,
  () => {
    if (sqlEditorStore.shouldFormatContent) {
      formatContent(editorInstance, language.value as SQLDialect);
      nextTick(() => {
        setPositionAtEndOfLine(editorInstance);
        editorInstance.focus();
      });
      sqlEditorStore.setShouldFormatContent(false);
    }
  }
);
</script>

<style>
.monaco-editor .cldr.sql-fragment {
  @apply bg-indigo-600;
  width: 3px !important;
  margin-left: 2px;
}
</style>
