"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { errorMessage } from "@/lib/api/client";
import { useAuth } from "@/providers/auth-provider";
import { Button } from "@/components/ui/button";
import { TextField } from "@/components/ui/form";

export default function RegisterPage() {
  const router = useRouter();
  const { register } = useAuth();
  const [form, setForm] = useState({ name: "", email: "", password: "", confirm: "" });
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const set = (field: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [field]: e.target.value }));

  const mismatch = form.confirm !== "" && form.confirm !== form.password;
  const tooShort = form.password !== "" && form.password.length < 8;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    if (mismatch || tooShort) return;
    setError(null);
    setLoading(true);
    try {
      await register({ name: form.name, email: form.email, password: form.password });
      router.replace("/");
    } catch (err) {
      setError(errorMessage(err));
      setLoading(false);
    }
  };

  return (
    <>
      <h1 className="text-2xl font-bold">Crear cuenta</h1>
      <form onSubmit={submit} className="mt-6 space-y-4">
        <TextField label="Nombre" value={form.name} onChange={set("name")} required autoComplete="name" />
        <TextField label="Correo electronico" type="email" value={form.email} onChange={set("email")} required autoComplete="email" />
        <TextField
          label="Contrasena"
          type="password"
          value={form.password}
          onChange={set("password")}
          required
          autoComplete="new-password"
          error={tooShort ? "Minimo 8 caracteres" : undefined}
          hint="Minimo 8 caracteres"
        />
        <TextField
          label="Confirmar contrasena"
          type="password"
          value={form.confirm}
          onChange={set("confirm")}
          required
          autoComplete="new-password"
          error={mismatch ? "Las contrasenas no coinciden" : undefined}
        />
        {error && (
          <p role="alert" className="rounded-lg bg-red-50 p-3 text-sm text-red-700">
            {error}
          </p>
        )}
        <Button type="submit" size="lg" className="w-full" loading={loading}>
          Crear cuenta
        </Button>
      </form>
      <p className="mt-6 text-center text-sm text-slate-500">
        ¿Ya tienes cuenta?{" "}
        <Link href="/login" className="font-medium text-brand-700 hover:underline">
          Iniciar sesion
        </Link>
      </p>
    </>
  );
}
