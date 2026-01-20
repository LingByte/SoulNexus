import { Language } from './common'

export const workflow: Record<Language, Record<string, string>> = {
  zh: {
    // Workflow 工作流
    'workflow.editor.addNode': '添加节点',
    'workflow.editor.help': '操作说明',
    'workflow.nodes.start': '开始',
    'workflow.nodes.end': '结束',
    'workflow.nodes.task': '任务',
    'workflow.nodes.gateway': '条件判断',
    'workflow.nodes.event': '事件',
    'workflow.nodes.subflow': '子流程',
    'workflow.nodes.parallel': '并行',
    'workflow.nodes.wait': '等待',
    'workflow.nodes.timer': '定时器',
    'workflow.nodes.script': '脚本',

    // Workflow Messages
    'workflow.messages.saveSuccess': '工作流保存成功',
    'workflow.messages.saveFailed': '保存工作流失败',
    'workflow.messages.deleteSuccess': '工作流删除成功',
    'workflow.messages.deleteFailed': '删除工作流失败',
    'workflow.messages.executeSuccess': '工作流执行成功',
    'workflow.messages.executeFailed': '工作流执行失败',
    'workflow.messages.fetchFailed': '获取工作流失败',
    'workflow.messages.createSuccess': '工作流创建成功',
    'workflow.messages.createFailed': '创建工作流失败',
  },
  en: {
    // Workflow
    'workflow.editor.addNode': 'Add Node',
    'workflow.editor.help': 'Help',
    'workflow.nodes.start': 'Start',
    'workflow.nodes.end': 'End',
    'workflow.nodes.task': 'Task',
    'workflow.nodes.gateway': 'Gateway',
    'workflow.nodes.event': 'Event',
    'workflow.nodes.subflow': 'Subflow',
    'workflow.nodes.parallel': 'Parallel',
    'workflow.nodes.wait': 'Wait',
    'workflow.nodes.timer': 'Timer',
    'workflow.nodes.script': 'Script',

    // Workflow Messages
    'workflow.messages.saveSuccess': 'Workflow saved successfully',
    'workflow.messages.saveFailed': 'Failed to save workflow',
    'workflow.messages.deleteSuccess': 'Workflow deleted successfully',
    'workflow.messages.deleteFailed': 'Failed to delete workflow',
    'workflow.messages.executeSuccess': 'Workflow executed successfully',
    'workflow.messages.executeFailed': 'Failed to execute workflow',
    'workflow.messages.fetchFailed': 'Failed to fetch workflow',
    'workflow.messages.createSuccess': 'Workflow created successfully',
    'workflow.messages.createFailed': 'Failed to create workflow',
  },
  ja: {
    // ワークフロー
    'workflow.editor.addNode': 'ノードを追加',
    'workflow.editor.help': 'ヘルプ',
    'workflow.nodes.start': '開始',
    'workflow.nodes.end': '終了',
    'workflow.nodes.task': 'タスク',
    'workflow.nodes.gateway': 'ゲートウェイ',
    'workflow.nodes.event': 'イベント',
    'workflow.nodes.subflow': 'サブフロー',
    'workflow.nodes.parallel': '並列',
    'workflow.nodes.wait': '待機',
    'workflow.nodes.timer': 'タイマー',
    'workflow.nodes.script': 'スクリプト',

    // ワークフローメッセージ
    'workflow.messages.saveSuccess': 'ワークフローが正常に保存されました',
    'workflow.messages.saveFailed': 'ワークフローの保存に失敗しました',
    'workflow.messages.deleteSuccess': 'ワークフローが正常に削除されました',
    'workflow.messages.deleteFailed': 'ワークフローの削除に失敗しました',
    'workflow.messages.executeSuccess': 'ワークフローが正常に実行されました',
    'workflow.messages.executeFailed': 'ワークフローの実行に失敗しました',
    'workflow.messages.fetchFailed': 'ワークフローの取得に失敗しました',
    'workflow.messages.createSuccess': 'ワークフローが正常に作成されました',
    'workflow.messages.createFailed': 'ワークフローの作成に失敗しました',
  }
}