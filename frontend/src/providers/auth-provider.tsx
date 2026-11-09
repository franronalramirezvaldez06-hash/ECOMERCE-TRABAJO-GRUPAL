"use client";

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { authApi } from "@/lib/api/services";
import { getToken, onUnauthorized, setToken } from "@/lib/api/client";
import type { User } from "@/lib/api/types";
import { useToast } from "./toast-provider";

interface AuthContextValue {
  user: User | null;
  /** true mientras se verifica el token guardado al cargar la pagina. */
  loading: boolean;
  isAdmin: boolean;
  login: (email: string, password: string) => Promise<User>;
  register: (input: { email: string; name: string; password: string }) => Promise<User>;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const queryClient = useQueryClient();
  const toast = useToast();

  const logout = useCallback(() => {
    setToken(null);
    setUser(null);
    queryClient.clear();
  }, [queryClient]);

  // Restaura la sesion si hay un token guardado.
  useEffect(() => {
    let cancelled = false;
    if (!getToken()) {
      setLoading(false);
      return;
    }
    authApi
      .me()
      .then((u) => !cancelled && setUser(u))
      .catch(() => setToken(null))
      .finally(() => !cancelled && setLoading(false));
    return () => {
      cancelled = true;
    };
  }, []);

  // Si el backend responde 401 con un token guardado, la sesion vencio.
  useEffect(() => {
    onUnauthorized(() => {
      logout();
      toast.info("Tu sesion expiro. Inicia sesion de nuevo.");
    });
    return () => onUnauthorized(null);
  }, [logout, toast]);

  const login = useCallback(async (email: string, password: string) => {
    const result = await authApi.login(email, password);
    setToken(result.token);
    setUser(result.user);
    return result.user;
  }, []);

  const register = useCallback(
    async (input: { email: string; name: string; password: string }) => {
      await authApi.register(input);
      return login(input.email, input.password);
    },
    [login],
  );

  const value = useMemo<AuthContextValue>(
    () => ({ user, loading, isAdmin: user?.role === "admin", login, register, logout }),
    [user, loading, login, register, logout],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth debe usarse dentro de AuthProvider");
  return ctx;
}
