"use server";

import { AuthError } from "next-auth";
import { z } from "zod";
import { signIn, signOut } from "@/auth";
import { t } from "@/i18n/id";

export type LoginState = { error?: string } | undefined;

const schema = z.object({
  email: z.string().email(),
  password: z.string().min(1),
  callbackUrl: z.string().optional(),
});

export async function loginAction(_prev: LoginState, formData: FormData): Promise<LoginState> {
  const parsed = schema.safeParse({
    email: formData.get("email"),
    password: formData.get("password"),
    callbackUrl: formData.get("callbackUrl") || undefined,
  });
  if (!parsed.success) return { error: t.auth.invalid };
  const redirectTo = parsed.data.callbackUrl && parsed.data.callbackUrl.startsWith("/") ? parsed.data.callbackUrl : "/";
  try {
    await signIn("credentials", {
      email: parsed.data.email,
      password: parsed.data.password,
      redirectTo,
    });
    return undefined;
  } catch (e) {
    if (e instanceof AuthError) {
      return { error: t.auth.invalid };
    }
    // next/navigation redirect() throws — rethrow so Next handles it
    throw e;
  }
}

export async function logoutAction(): Promise<void> {
  await signOut({ redirectTo: "/login" });
}
