<template>
  <MonacoEditor
    v-model:value="sqlCode"
    :language="selectedLanguage"
    @change="handleChange"
    @change-selection="handleChangeSelection"
    @run-query="handleRunQuery"
    @save="(e) => emit('save-sheet')"
  />
</template>

<script lang="ts" setup>
import { debounce } from "lodash-es";
import { computed, defineEmits } from "vue";

import { useInstanceStore, useTabStore, useSQLEditorStore } from "@/store";
import { useExecuteSQL } from "@/composables/useExecuteSQL";

const emit = defineEmits<{
  (e: "save-sheet", content?: string): void;
}>();

const instanceStore = useInstanceStore();
const tabStore = useTabStore();
const sqlEditorStore = useSQLEditorStore();

const { execute } = useExecuteSQL();

const sqlCode = computed(() => tabStore.currentTab.statement);
const selectedInstance = computed(() => {
  const ctx = sqlEditorStore.connectionContext;
  return instanceStore.getInstanceById(ctx.instanceId);
});
const selectedInstanceEngine = computed(() => {
  return instanceStore.formatEngine(selectedInstance.value);
});

const selectedLanguage = computed(() => {
  const engine = selectedInstanceEngine.value;
  if (engine === "MySQL") {
    return "mysql";
  }
  if (engine === "PostgreSQL") {
    return "pgsql";
  }
  return "sql";
});

const handleChange = debounce((value: string) => {
  tabStore.updateCurrentTab({
    statement: value,
    isSaved: false,
  });
}, 300);

const handleChangeSelection = debounce((value: string) => {
  tabStore.updateCurrentTab({
    selectedStatement: value,
  });
}, 300);

const handleRunQuery = ({
  explain,
  query,
}: {
  explain: boolean;
  query: string;
}) => {
  execute({ databaseType: selectedInstanceEngine.value }, { explain });
};
</script>
