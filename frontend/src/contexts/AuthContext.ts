import { createContext } from 'react';
import { User, PutV1SettingsMutationBody as UserSettings } from '../api/api';

export interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (username: string, password: string) => Promise<boolean>;
  loginWithUser: (userData: User) => Promise<boolean>;
  logout: () => Promise<void>;
  updateSettings: (settings: UserSettings) => Promise<boolean>;
  refreshUser: () => Promise<void>;
}

export const AuthContext = createContext<AuthContextType | undefined>(
  undefined
);
