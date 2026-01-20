import React, { createContext, useContext, useEffect, ReactNode } from 'react';
import { useAuthStore, type User } from '@/stores/authStore';

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (token: string) => Promise<boolean>;
  register: (data: any) => Promise<boolean>;
  logout: (next?: string) => Promise<void>;
  setLoading: (loading: boolean) => void;
  refreshUserInfo: () => Promise<void>;
  updateProfile: (data: Partial<User>) => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const authStore = useAuthStore();

  useEffect(() => {
    // 初始化时检查是否有token
    const token = localStorage.getItem('auth_token');
    if (token) {
      authStore.refreshUserInfo();
    }
  }, []);

  const value: AuthContextType = {
    user: authStore.user,
    isAuthenticated: authStore.isAuthenticated,
    isLoading: authStore.isLoading,
    login: authStore.login,
    register: authStore.register,
    logout: authStore.logout,
    setLoading: authStore.setLoading,
    refreshUserInfo: authStore.refreshUserInfo,
    updateProfile: authStore.updateProfile,
  };

  return (
    <AuthContext.Provider value={value}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
