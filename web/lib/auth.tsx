"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useRouter } from "next/navigation";
import {
  loginRequest,
  logoutRequest,
  meRequest,
  registerRequest,
  setUnauthorizedHandler,
  type User,
} from "@/lib/api";

type AuthContextValue = {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, password: string, name: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const clearSession = useCallback(() => {
    setUser(null);
  }, []);

  const refreshUser = useCallback(async () => {
    try {
      const profile = await meRequest();
      setUser(profile);
    } catch {
      clearSession();
    }
  }, [clearSession]);

  useEffect(() => {
    setUnauthorizedHandler(() => {
      clearSession();
      router.replace("/login/");
    });
    return () => setUnauthorizedHandler(null);
  }, [clearSession, router]);

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const profile = await meRequest();
        if (active) {
          setUser(profile);
        }
      } catch {
        if (active) {
          clearSession();
        }
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    })();
    return () => {
      active = false;
    };
  }, [clearSession]);

  const login = useCallback(
    async (email: string, password: string) => {
      const res = await loginRequest(email, password);
      setUser(res.user);
    },
    [],
  );

  const register = useCallback(
    async (email: string, password: string, name: string) => {
      const res = await registerRequest(email, password, name);
      setUser(res.user);
    },
    [],
  );

  const logout = useCallback(async () => {
    try {
      await logoutRequest();
    } finally {
      clearSession();
      router.replace("/login/");
    }
  }, [clearSession, router]);

  const value = useMemo(
    () => ({
      user,
      loading,
      login,
      register,
      logout,
      refreshUser,
    }),
    [user, loading, login, register, logout, refreshUser],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) {
    throw new Error("useAuth must be used within AuthProvider");
  }
  return ctx;
}
