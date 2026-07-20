export function chunkStrategyLabel(t: (key: string) => string, strategy?: string): string {
  switch (strategy) {
    case 'rules_table_kv':
      return t('knowledgeBase.doc.strategyRulesTableKV')
    case 'llm':
      return t('knowledgeBase.doc.strategyLLM')
    case 'rules_structured':
      return t('knowledgeBase.doc.strategyRulesStructured')
    default:
      return strategy || t('knowledgeBase.doc.strategyRulesStructured')
  }
}

export function recallStrategyLabel(t: (key: string) => string, strategy?: string): string {
  switch (strategy) {
    case 'hybrid':
      return t('knowledgeBase.recall.strategyHybrid')
    case 'keyword':
      return t('knowledgeBase.recall.strategyKeyword')
    case 'vector':
      return t('knowledgeBase.recall.strategyVector')
    default:
      return strategy || t('knowledgeBase.recall.strategyHybrid')
  }
}
