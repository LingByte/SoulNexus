import { create } from 'zustand';
import type { MCPServer, MCPServerDetail, MCPCallLog, MCPState } from '@/types/mcp';
import * as mcpAPI from '@/api/mcp';

interface MCPStoreState extends MCPState {
  // 操作
  fetchServers: () => Promise<void>;
  fetchServerDetail: (id: number) => Promise<void>;
  fetchLogs: (id: number) => Promise<void>;
  setSelectedServer: (server: MCPServerDetail | null) => void;
  setError: (error: string | null) => void;
  setLoading: (loading: boolean) => void;
}

export const useMCPStore = create<MCPStoreState>((set) => ({
  servers: [],
  selectedServer: null,
  loading: false,
  error: null,
  logs: [],

  fetchServers: async () => {
    set({ loading: true, error: null });
    try {
      const response = await mcpAPI.listMCPServers();
      if (response.code === 200) {
        set({ servers: response.data });
      } else {
        set({ error: 'Failed to fetch MCP servers' });
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : 'Unknown error' });
    } finally {
      set({ loading: false });
    }
  },

  fetchServerDetail: async (id: number) => {
    set({ loading: true, error: null });
    try {
      const response = await mcpAPI.getMCPServer(id);
      if (response.code === 200) {
        set({ selectedServer: response.data });
      } else {
        set({ error: 'Failed to fetch MCP server detail' });
      }
    } catch (err) {
      set({ error: err instanceof Error ? err.message : 'Unknown error' });
    } finally {
      set({ loading: false });
    }
  },

  fetchLogs: async (id: number) => {
    try {
      const response = await mcpAPI.getMCPLogs(id, 100);
      if (response.code === 200) {
        set({ logs: response.data });
      }
    } catch (err) {
      console.error('Failed to fetch logs:', err);
    }
  },

  setSelectedServer: (server: MCPServerDetail | null) => {
    set({ selectedServer: server });
  },

  setError: (error: string | null) => {
    set({ error });
  },

  setLoading: (loading: boolean) => {
    set({ loading });
  },
}));
