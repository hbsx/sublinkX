import request from "@/utils/request";
import type { AxiosPromise } from "axios";

export interface LoginRecord {
  id: number;
  time: string;
  username: string;
  ip: string;
  ipSource: string;
  userAgent: string;
  success: boolean;
  reason: string;
}

export interface LoginHistory {
  list: LoginRecord[];
  limit: number;
  startedAt: string;
  notifications: { sent: number; failed: number; dropped: number; lastSentAt: string; lastError: string };
}

export interface LoginNotificationSettings {
  enabled: boolean;
  botTokenConfigured: boolean;
  chatId: string;
  proxyUrl: string;
}

export const getLoginHistory = (): AxiosPromise<LoginHistory> =>
  request({ url: "/api/v1/users/login-history", method: "get" });

export const getLoginNotificationSettings = (): AxiosPromise<LoginNotificationSettings> =>
  request({ url: "/api/v1/users/login-notifications", method: "get" });

export const saveLoginNotificationSettings = (data: {
  enabled: boolean; botToken: string; chatId: string; proxyUrl: string;
}): AxiosPromise<LoginNotificationSettings> =>
  request({ url: "/api/v1/users/login-notifications", method: "put", data });

export const testLoginNotification = () =>
  request({ url: "/api/v1/users/login-notifications/test", method: "post" });
