import { AccountDetailPage } from "@/features/banking/account-pages";

export default async function Page({
  params,
}: {
  params: Promise<{ accountId: string }>;
}) {
  const { accountId } = await params;
  return <AccountDetailPage accountID={accountId} />;
}
