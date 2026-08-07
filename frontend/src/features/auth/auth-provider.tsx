"use client";

import { useQueryClient } from "@tanstack/react-query";
import {
  createContext,
  use,
  useCallback,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

import {
  clearAccessCredential,
  getCurrentUser,
  loginUser,
  logoutAllSessions,
  logoutCurrentSession,
  renewAccessToken,
  sessionExpiredEvent,
  updateCurrentUser,
  type LoginInput,
  type UpdateUserInput,
} from "@/features/auth/auth-api";
import { isApiError } from "@/lib/api/api-error";
import type { User } from "@/lib/api/contracts";
import { queryKeys } from "@/lib/query/query-keys";

type SessionStatus =
  "restoring" | "authenticated" | "anonymous" | "unavailable";

type AuthContextValue = {
  login: (input: LoginInput) => Promise<User>;
  logout: () => Promise<void>;
  logoutAll: () => Promise<void>;
  refreshUser: () => Promise<User>;
  restore: () => Promise<User | null>;
  status: SessionStatus;
  updateUser: (input: UpdateUserInput) => Promise<User>;
  user: User | null;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [status, setStatus] = useState<SessionStatus>("restoring");
  const [user, setUser] = useState<User | null>(null);
  const restoreInFlight = useRef<Promise<User | null> | null>(null);

  const restore = useCallback(async () => {
    if (!restoreInFlight.current) {
      restoreInFlight.current = renewAccessToken()
        .then(getCurrentUser)
        .then((currentUser) => {
          setUser(currentUser);
          setStatus("authenticated");
          queryClient.setQueryData(queryKeys.currentUser, currentUser);
          return currentUser;
        })
        .catch((error: unknown) => {
          clearAccessCredential();
          setUser(null);
          if (isApiError(error) && error.status === 401) {
            setStatus("anonymous");
            return null;
          }
          setStatus("unavailable");
          return null;
        })
        .finally(() => {
          restoreInFlight.current = null;
        });
    }
    return restoreInFlight.current;
  }, [queryClient]);

  useEffect(() => {
    void restore();
  }, [restore]);

  useEffect(() => {
    function handleSessionExpired() {
      clearAccessCredential();
      setUser(null);
      setStatus("anonymous");
      queryClient.clear();
    }

    window.addEventListener(sessionExpiredEvent, handleSessionExpired);
    return () =>
      window.removeEventListener(sessionExpiredEvent, handleSessionExpired);
  }, [queryClient]);

  async function login(input: LoginInput) {
    const currentUser = await loginUser(input);
    setUser(currentUser);
    setStatus("authenticated");
    queryClient.setQueryData(queryKeys.currentUser, currentUser);
    return currentUser;
  }

  async function refreshUser() {
    const currentUser = await getCurrentUser();
    setUser(currentUser);
    queryClient.setQueryData(queryKeys.currentUser, currentUser);
    return currentUser;
  }

  async function updateUser(input: UpdateUserInput) {
    const currentUser = await updateCurrentUser(input);
    setUser(currentUser);
    queryClient.setQueryData(queryKeys.currentUser, currentUser);
    return currentUser;
  }

  function clearPrivateState() {
    setUser(null);
    setStatus("anonymous");
    queryClient.clear();
  }

  async function logout() {
    try {
      await logoutCurrentSession();
    } finally {
      clearPrivateState();
    }
  }

  async function logoutAll() {
    try {
      await logoutAllSessions();
    } finally {
      clearPrivateState();
    }
  }

  return (
    <AuthContext
      value={{
        login,
        logout,
        logoutAll,
        refreshUser,
        restore,
        status,
        updateUser,
        user,
      }}
    >
      {children}
    </AuthContext>
  );
}

export function useAuth() {
  const context = use(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return context;
}
