import { CloseAccountPage } from "@/features/banking/account-pages";

export default async function Page({
  params,
}: {
  params: Promise<{ accountId: string }>;
}) {
  const { accountId } = await params;
  return <CloseAccountPage accountID={accountId} />;
}
