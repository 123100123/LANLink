export type HealthResponse = {
  status: string;
  service: string;
};

export type PairRequest = {
  device_name: string;
  token: string;
};

export type PairResponse = {
  status: string;
  device_id?: string;
  auth_token?: string;
  error?: string;
};

export type DeviceInfo = {
  device_id: string;
  device_name: string;
};

export type DevicesResponse = {
  devices: DeviceInfo[];
};

export type AuthRequest = {
  token: string;
};

export type AuthSuccess = {
  device_id: string;
  device_name: string;
};

export type AuthFailed = {
  error: string;
};

export type PingPayload = {
  sent_at: number;
};

export type PongPayload = {
  sent_at: number;
  received_at: number;
};

export type DirectMessagePayload = {
  text: string;
};

export type DirectMessageResponse = {
  status: string;
};

export type LanLinkMessage = {
  type: string;
  id?: string;
  module?: string;
  action?: string;
  timestamp?: number;
  payload?: unknown;
};
