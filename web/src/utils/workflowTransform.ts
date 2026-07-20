import type { WorkflowDefinition, WorkflowGraph, WorkflowNodeType, WorkflowEdgeType, WorkflowNodeSchema } from '@/api/workflow'
import type { Workflow as EditorWorkflow } from '@/components/workflow/types'

function getInputsOutputs(type: WorkflowNodeType) {
  switch (type) {
    case 'start':
      return { inputs: 0, outputs: 1 }
    case 'end':
      return { inputs: 1, outputs: 0 }
    case 'gateway':
    case 'parallel':
      return { inputs: 1, outputs: 2 }
    case 'event':
      return { inputs: 0, outputs: 1 }
    default:
      return { inputs: 1, outputs: 1 }
  }
}

function kbInputVariable(node: EditorWorkflow['nodes'][number]): string {
  return String(node.data.config?.inputVariable || 'query')
}

function kbOutputVariable(node: EditorWorkflow['nodes'][number]): string {
  return String(node.data.config?.outputVariable || 'kb_result')
}

function aiInputVariable(node: EditorWorkflow['nodes'][number]): string {
  return String(node.data.config?.inputVariable || 'user_input')
}

function aiOutputVariable(node: EditorWorkflow['nodes'][number]): string {
  return String(node.data.config?.outputVariable || 'ai_response')
}

export function toEditorWorkflow(workflow: WorkflowDefinition): EditorWorkflow {
  return {
    id: workflow.id.toString(),
    name: workflow.name,
    description: workflow.description,
    nodes: workflow.definition.nodes.map((n) => {
      const nodeType = n.type as EditorWorkflow['nodes'][number]['type']
      const { inputs: inputCount, outputs: outputCount } = getInputsOutputs(n.type)
      const inputKeys = Object.keys(n.inputMap || {})
      const outputKeys = Object.keys(n.outputMap || {})
      const pluginIdStr = n.properties?.pluginId || n.properties?.['pluginId']
      const pluginId = pluginIdStr
        ? (typeof pluginIdStr === 'string' ? parseInt(pluginIdStr, 10) : pluginIdStr)
        : undefined

      let inputs = inputKeys.length > 0
        ? inputKeys
        : Array(inputCount).fill('').map((_, i) => `input-${i}`)
      let outputs = outputKeys.length > 0
        ? outputKeys
        : Array(outputCount).fill('').map((_, i) => `output-${i}`)

      if (nodeType === 'knowledge_base') {
        const inputVar = n.properties?.inputVariable || 'query'
        const outputVar = n.properties?.outputVariable || 'kb_result'
        inputs = [inputVar]
        outputs = [outputVar]
      } else if (nodeType === 'ai_chat') {
        const inputVar = n.properties?.inputVariable || 'user_input'
        const outputVar = n.properties?.outputVariable || 'ai_response'
        inputs = [inputVar]
        outputs = [outputVar]
      }

      return {
        id: n.id,
        type: nodeType,
        position: n.position || { x: 0, y: 0 },
        data: {
          label: n.name,
          config: n.properties || {},
          _inputMap: n.inputMap,
          _outputMap: n.outputMap,
          ...(n.type === 'workflow_plugin' && pluginId ? { pluginId } : {}),
        },
        inputs,
        outputs,
      }
    }),
    connections: workflow.definition.edges.map((e) => {
      let sourceHandle = 'output-0'
      const sourceNode = workflow.definition.nodes.find((n) => n.id === e.source)
      if (sourceNode) {
        if (sourceNode.type === 'condition' || sourceNode.type === 'gateway') {
          if (e.type === 'true') sourceHandle = 'output-0'
          else if (e.type === 'false') sourceHandle = 'output-1'
        } else if (sourceNode.type === 'parallel' && e.type === 'branch') {
          const branchIndex = workflow.definition.edges
            .filter((edge) => edge.source === e.source && edge.type === 'branch')
            .findIndex((edge) => edge.id === e.id)
          sourceHandle = `output-${branchIndex}`
        }
      }

      return {
        id: e.id,
        source: e.source,
        target: e.target,
        sourceHandle,
        targetHandle: 'input-0',
        type: e.type || 'default',
        condition: e.condition,
      }
    }),
    createdAt: workflow.createdAt,
    updatedAt: workflow.updatedAt,
  }
}

/** 将画布编辑器数据转换回后端 WorkflowGraph */
export function editorWorkflowToGraph(workflow: EditorWorkflow): WorkflowGraph {
  return {
    nodes: workflow.nodes.map((node) => {
      const inputMap: Record<string, string> = {}
      const outputMap: Record<string, string> = {}

      if (node.type === 'start') {
        node.inputs?.forEach((input) => {
          if (input?.trim()) inputMap[input] = input
        })
      } else if (node.type === 'end') {
        node.outputs?.forEach((output) => {
          if (output?.trim()) {
            const incomingEdge = workflow.connections.find((conn) => conn.target === node.id)
            if (incomingEdge) {
              const sourceNode = workflow.nodes.find((n) => n.id === incomingEdge.source)
              if (sourceNode?.type === 'ai_chat') {
                const outputVar = sourceNode.data.config?.outputVariable
                outputMap[output] = outputVar ? `${sourceNode.id}.${outputVar}` : output
              } else if (sourceNode?.type === 'knowledge_base') {
                const outputVar = sourceNode.data.config?.outputVariable || 'kb_result'
                outputMap[output] = `${sourceNode.id}.${outputVar}`
              } else if (sourceNode?.type === 'workflow_plugin' || sourceNode?.type === 'script') {
                outputMap[output] = `${sourceNode.id}.${output}`
              } else {
                outputMap[output] = output
              }
            } else {
              outputMap[output] = output
            }
          }
        })
      } else {
        const inputVar =
          node.type === 'knowledge_base'
            ? kbInputVariable(node)
            : node.type === 'ai_chat'
              ? aiInputVariable(node)
              : ''
        const outputVar =
          node.type === 'knowledge_base'
            ? kbOutputVariable(node)
            : node.type === 'ai_chat'
              ? aiOutputVariable(node)
              : ''

        node.inputs?.forEach((input) => {
          if (input?.trim()) {
            const incomingEdge = workflow.connections.find((conn) => conn.target === node.id)
            if (incomingEdge) {
              const sourceNode = workflow.nodes.find((n) => n.id === incomingEdge.source)
              if (sourceNode) {
                if (sourceNode.type === 'workflow_plugin') {
                  const sourceOutputMap = sourceNode.data._outputMap || sourceNode.data.config?.outputMap || {}
                  let sourceOutput = input
                  for (const [key, value] of Object.entries(sourceOutputMap)) {
                    if (key === input || value === input) {
                      sourceOutput = key
                      break
                    }
                  }
                  if (!sourceOutput && sourceNode.outputs?.length) {
                    sourceOutput = sourceNode.outputs[0]
                  }
                  inputMap[input] = `${sourceNode.id}.${sourceOutput}`
                } else if (sourceNode.type === 'ai_chat') {
                  const srcOut = sourceNode.data.config?.outputVariable || 'ai_response'
                  inputMap[input] = `${sourceNode.id}.${srcOut}`
                } else if (sourceNode.type === 'knowledge_base') {
                  const srcOut = sourceNode.data.config?.outputVariable || 'kb_result'
                  inputMap[input] = `${sourceNode.id}.${srcOut}`
                } else {
                  inputMap[input] = `${sourceNode.id}.${input}`
                }
              } else {
                inputMap[input] = inputVar || input
              }
            } else {
              inputMap[inputVar || input] = inputVar || input
            }
          }
        })

        node.outputs?.forEach((output) => {
          if (output?.trim()) {
            const outKey = outputVar || output
            outputMap[output] = `${node.id}.${outKey}`
          }
        })
      }

      const properties: Record<string, string> = {}
      if (node.data.config) {
        for (const [key, value] of Object.entries(node.data.config)) {
          if (value !== null && value !== undefined) {
            if (key === 'parameters' && typeof value === 'object') {
              properties[key] = JSON.stringify(value)
            } else {
              properties[key] = String(value)
            }
          }
        }
      }

      return {
        id: node.id,
        name: node.data.label,
        type: node.type as WorkflowNodeType,
        position: node.position,
        properties,
        inputMap,
        outputMap,
      }
    }),
    edges: workflow.connections.map((conn) => {
      let edgeType: WorkflowEdgeType = (conn.type as WorkflowEdgeType | undefined) || 'default'
      const sourceNode = workflow.nodes.find((n) => n.id === conn.source)
      if (sourceNode?.type === 'gateway') {
        const outputIndex = sourceNode.outputs.findIndex((o) => o === conn.sourceHandle)
        if (outputIndex === 0) edgeType = 'true'
        else if (outputIndex === 1) edgeType = 'false'
      } else if (sourceNode?.type === 'parallel') {
        edgeType = 'branch'
      }

      return {
        id: conn.id,
        source: conn.source,
        target: conn.target,
        type: edgeType,
        condition: conn.condition || undefined,
      }
    }),
  }
}

/** 将单个画布节点转为后端 Schema（用于未保存前的单节点测试） */
export function editorNodeToSchema(
  node: EditorWorkflow['nodes'][number],
  allNodes: EditorWorkflow['nodes'],
  connections: EditorWorkflow['connections'],
): WorkflowNodeSchema {
  const graph = editorWorkflowToGraph({
    id: 'draft',
    name: '',
    description: '',
    nodes: allNodes,
    connections,
    createdAt: '',
    updatedAt: '',
  })
  const schema = graph.nodes.find((n) => n.id === node.id)
  if (!schema) {
    throw new Error(`node ${node.id} not found in graph`)
  }
  return schema
}
