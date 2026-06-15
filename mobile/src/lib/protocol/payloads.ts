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

export type FileStartPayload = {
  transfer_id: string;
  filename: string;
  size: number;
};

export type FileChunkPayload = {
  transfer_id: string;
  index: number;
  offset: number;
  length: number;
  content: string;
};

export type FileEndPayload = {
  transfer_id: string;
};

export type FileChunkResponse = {
  status: string;
  transfer_id?: string;
  index?: number;
  offset?: number;
  received?: number;
  total?: number;
  path?: string;
  error?: string;
};

export type LanLinkMessage = {
  type: string;
  id?: string;
  module?: string;
  action?: string;
  timestamp?: number;
  payload?: unknown;
};

export type TransferStartRequest = {
  transfer_id: string;
  filename: string;
  size: number;
};

export type TransferStartResponse = {
  status: string;
  transfer_id?: string;
  total?: number;
  error?: string;
};

export type TransferChunkResponse = {
  status: string;
  transfer_id?: string;
  index?: number;
  offset?: number;
  received?: number;
  total?: number;
  error?: string;
};

export type TransferFinishResponse = {
  status: string;
  transfer_id?: string;
  path?: string;
  received?: number;
  total?: number;
  error?: string;
};
