"use client";

import { ShieldCheck, UserRound } from "lucide-react";
import type { Route } from "next";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { PasswordInput } from "@/components/auth/password-input";
import { authErrorMessage } from "@/features/auth/auth-errors";
import { useAuth } from "@/features/auth/auth-provider";
import {
  emailSchema,
  fullNameSchema,
  passwordSchema,
} from "@/features/auth/auth-schemas";

export function ProfilePage() {
  const { user } = useAuth();
  if (!user) return null;
  return (
    <div className="max-w-3xl">
      <header className="border-line-200 flex flex-col gap-5 border-b pb-7 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-4xl font-semibold">Profile</h1>
          <p className="text-ink-600 mt-3">
            Review the personal information connected to your account.
          </p>
        </div>
        <Button asChild>
          <Link href={"/app/profile/edit" as Route}>Edit profile</Link>
        </Button>
      </header>
      <dl className="mt-8 grid gap-6">
        <ProfileValue label="Full name" value={user.full_name} />
        <ProfileValue label="Username" value={`@${user.username}`} />
        <ProfileValue label="Email address" value={user.email} />
        <ProfileValue
          label="Email verification"
          value={user.email_verified_at ? "Verified" : "Not verified"}
        />
        <ProfileValue label="Account status" value={user.account_status} />
        <ProfileValue
          label="Member since"
          value={new Date(user.created_at).toLocaleDateString()}
        />
      </dl>
    </div>
  );
}

export function EditProfilePage() {
  const { updateUser, user } = useAuth();
  const router = useRouter();
  const [fullName, setFullName] = useState(user?.full_name ?? "");
  const [email, setEmail] = useState(user?.email ?? "");
  const [emailCurrentPassword, setEmailCurrentPassword] = useState("");
  const [currentPassword, setCurrentPassword] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState<string>();
  const [notice, setNotice] = useState<string>();
  const [savingProfile, setSavingProfile] = useState(false);
  const [savingPassword, setSavingPassword] = useState(false);

  async function saveProfile() {
    const name = fullNameSchema.safeParse(fullName);
    const address = emailSchema.safeParse(email);
    if (!name.success || !address.success) {
      setError(
        name.error?.issues[0]?.message ?? address.error?.issues[0]?.message,
      );
      return;
    }
    setSavingProfile(true);
    setError(undefined);
    try {
      const changedEmail = address.data !== user?.email;
      if (changedEmail && !emailCurrentPassword) {
        setError("Enter your current password to change your email address.");
        return;
      }
      await updateUser({
        email: address.data,
        full_name: name.data,
        ...(changedEmail
          ? { current_password: emailCurrentPassword }
          : undefined),
      });
      if (changedEmail) {
        router.replace("/signup/check-email");
      } else {
        setNotice("Profile updated.");
      }
    } catch (cause) {
      setError(authErrorMessage(cause, "update-email"));
    } finally {
      setSavingProfile(false);
    }
  }

  async function savePassword() {
    const nextPassword = passwordSchema.safeParse(password);
    if (!currentPassword) {
      setError("Enter your current password.");
      return;
    }
    if (!nextPassword.success) {
      setError(nextPassword.error.issues[0]?.message);
      return;
    }
    setSavingPassword(true);
    setError(undefined);
    try {
      await updateUser({
        current_password: currentPassword,
        password: nextPassword.data,
      });
      router.replace("/login");
    } catch (cause) {
      setError(authErrorMessage(cause, "signup"));
    } finally {
      setSavingPassword(false);
    }
  }

  return (
    <div className="max-w-2xl">
      <Link
        className="text-evergreen-800 inline-flex min-h-11 items-center font-semibold no-underline"
        href={"/app/profile" as Route}
      >
        Back to profile
      </Link>
      <h1 className="mt-5 text-4xl font-semibold">Edit profile</h1>
      <p className="text-ink-600 mt-3">
        Changing your email requires verification before financial features
        become available again.
      </p>
      {error || notice ? (
        <div
          className={`mt-6 rounded-sm border-l-4 p-4 ${
            error
              ? "border-danger-700 bg-[#fff5f3]"
              : "border-success-700 bg-jade-100"
          }`}
          role="status"
        >
          {error ?? notice}
        </div>
      ) : null}
      <section className="border-line-200 mt-8 border-b pb-8">
        <h2 className="text-xl font-semibold">Personal information</h2>
        <div className="mt-5 grid gap-5">
          <Field label="Full name" name="profile-name">
            <Input
              autoComplete="name"
              id="profile-name"
              onChange={(event) => setFullName(event.target.value)}
              value={fullName}
            />
          </Field>
          <Field label="Email address" name="profile-email">
            <Input
              autoComplete="email"
              id="profile-email"
              onChange={(event) => setEmail(event.target.value)}
              type="email"
              value={email}
            />
          </Field>
          {email.trim().toLowerCase() !== user?.email ? (
            <Field
              hint="For your security, changing your email signs you out on every device."
              label="Current password"
              name="profile-current-password"
            >
              <PasswordInput
                autoComplete="current-password"
                id="profile-current-password"
                onChange={(event) =>
                  setEmailCurrentPassword(event.target.value)
                }
                value={emailCurrentPassword}
              />
            </Field>
          ) : null}
          <Button
            className="sm:justify-self-start"
            loading={savingProfile}
            onClick={() => void saveProfile()}
          >
            Save profile
          </Button>
        </div>
      </section>
      <section className="mt-8">
        <h2 className="text-xl font-semibold">Change password</h2>
        <p className="text-ink-600 mt-2 text-sm">
          Changing your password revokes your other sessions.
        </p>
        <div className="mt-5 grid gap-5">
          <Field label="Current password" name="current-password">
            <PasswordInput
              autoComplete="current-password"
              id="current-password"
              onChange={(event) => setCurrentPassword(event.target.value)}
              value={currentPassword}
            />
          </Field>
          <Field
            hint="Use 8–72 bytes. Known breached passwords are rejected."
            label="New password"
            name="new-password"
          >
            <PasswordInput
              autoComplete="new-password"
              id="new-password"
              onChange={(event) => setPassword(event.target.value)}
              value={password}
            />
          </Field>
          <Button
            className="sm:justify-self-start"
            loading={savingPassword}
            onClick={() => void savePassword()}
          >
            Change password
          </Button>
        </div>
      </section>
    </div>
  );
}

export function SecurityPage() {
  const { logoutAll } = useAuth();
  const router = useRouter();
  const [confirming, setConfirming] = useState(false);
  const [loading, setLoading] = useState(false);
  return (
    <div className="max-w-3xl">
      <ShieldCheck aria-hidden="true" className="text-evergreen-700 size-9" />
      <h1 className="mt-4 text-4xl font-semibold">Security</h1>
      <p className="text-ink-600 mt-3 max-w-2xl leading-7">
        Monierave starts a fresh exclusive session after successful login,
        rotates refresh credentials, and stores access credentials only in
        memory.
      </p>
      <section className="border-line-200 mt-9 border-y py-6">
        <div className="flex items-start gap-4">
          <UserRound
            aria-hidden="true"
            className="text-evergreen-700 mt-1 size-6"
          />
          <div>
            <h2 className="text-xl font-semibold">Sign out everywhere</h2>
            <p className="text-ink-600 mt-2 leading-6">
              End the current session and revoke every other session associated
              with your account.
            </p>
            {confirming ? (
              <div className="mt-5">
                <p className="font-semibold">
                  You will need to sign in again on this device.
                </p>
                <div className="mt-3 flex gap-2">
                  <Button
                    loading={loading}
                    onClick={async () => {
                      setLoading(true);
                      try {
                        await logoutAll();
                      } finally {
                        router.replace("/login");
                      }
                    }}
                    variant="danger"
                  >
                    Sign out everywhere
                  </Button>
                  <Button
                    onClick={() => setConfirming(false)}
                    variant="secondary"
                  >
                    Cancel
                  </Button>
                </div>
              </div>
            ) : (
              <Button
                className="mt-5"
                onClick={() => setConfirming(true)}
                variant="secondary"
              >
                Review logout
              </Button>
            )}
          </div>
        </div>
      </section>
      <p className="text-ink-600 mt-6 text-sm leading-6">
        Individual device lists, MFA, and step-up authentication are not shown
        because the backend does not yet provide those security controls.
      </p>
    </div>
  );
}

function ProfileValue({ label, value }: { label: string; value: string }) {
  return (
    <div className="border-line-200 grid gap-1 border-b pb-4 sm:grid-cols-[12rem_1fr]">
      <dt className="text-ink-600">{label}</dt>
      <dd className="font-semibold capitalize">{value}</dd>
    </div>
  );
}
