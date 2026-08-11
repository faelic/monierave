export type NullableTimestamp = string | null;

export type User = {
  username: string;
  full_name: string;
  email: string;
  email_verified_at: NullableTimestamp;
  email_deliverability_status: string;
  email_bounced_at: NullableTimestamp;
  account_status: "pending" | "active" | "disabled";
  registration_expires_at: NullableTimestamp;
  password_changed_at: string;
  created_at: string;
};

export type Account = {
  id: string;
  account_number: string;
  currency: "USD" | "EUR";
  balance: number;
  status: "active" | "frozen" | "closed";
  created_at: string;
  updated_at: string;
  closed_at: NullableTimestamp;
};

export type RecipientResolution = {
  account_number: string;
  account_name: string;
  currency: Account["currency"];
  can_receive: boolean;
};

export type Beneficiary = {
  id: string;
  nickname: string;
  destination_account_number: string;
  currency: Account["currency"];
  can_receive: boolean;
  created_at: string;
  updated_at: string;
};

export type BankingTransaction = {
  id: string;
  reference: string;
  account_id: string;
  type: "deposit" | "withdrawal" | "internal_transfer" | "reversal";
  status: "pending" | "posted" | "failed" | "reversed";
  currency: Account["currency"];
  amount: number;
  direction: "incoming" | "outgoing";
  narration: string;
  counterparty_type: string;
  counterparty: string;
  balance_after?: number;
  created_at: string;
  posted_at: NullableTimestamp;
};

export type TransactionHistoryPage = {
  transactions: BankingTransaction[];
  next_cursor?: string;
};

export type TransferDestination = {
  account_number: string;
  currency: Account["currency"];
};

export type TransferResponse = {
  transaction: TransferTransaction;
  from_account: Account;
  to_account: TransferDestination;
  fee: 0;
};

export type TransferTransaction = {
  id: string;
  reference: string;
  type: BankingTransaction["type"];
  status: BankingTransaction["status"];
  currency: Account["currency"];
  amount: number;
  narration: string;
  created_at: string;
  posted_at: NullableTimestamp;
};

export type AccountStatement = {
  account_id: string;
  currency: Account["currency"];
  from: NullableTimestamp;
  to: NullableTimestamp;
  opening_balance: number;
  closing_balance: number;
  transactions: BankingTransaction[];
  next_cursor?: string;
};

export type ApiErrorEnvelope = {
  code: string;
  message: string;
  error?: string;
  request_id?: string;
};

export type LoginResponse = {
  access_token: string;
  access_token_expires_at: string;
  user: User;
};

export type RenewAccessResponse = {
  access_token: string;
  access_token_expires_at: string;
};

export type EmailJobStatus = {
  id: string;
  worker_status: string;
  delivery_status: string;
  provider_message_id: string | null;
  accepted_at: NullableTimestamp;
  delivered_at: NullableTimestamp;
  bounced_at: NullableTimestamp;
  bounce_type: string | null;
  bounce_subtype: string | null;
};

export type EmailStatus = {
  email: string;
  verified_at: NullableTimestamp;
  deliverability_status: string;
  bounced_at: NullableTimestamp;
  latest_job?: EmailJobStatus;
  account_status: User["account_status"];
  registration_expires_at: NullableTimestamp;
  allowed_features?: string[];
  restricted_features?: string[];
};
