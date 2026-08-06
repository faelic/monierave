import { TransactionDetailPage } from "@/features/banking/account-pages";

export default async function Page({
  params,
}: {
  params: Promise<{ reference: string }>;
}) {
  const { reference } = await params;
  return <TransactionDetailPage reference={reference} />;
}
