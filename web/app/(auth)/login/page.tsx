import Link from "next/link";
import { Button } from "@/components/ui/Button";

export default function LoginPage() {
  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <div className="w-full max-w-md">
        <h1 className="mb-8 text-center text-2xl font-bold text-slate-900">Kanba</h1>
        <div className="rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
          <h2 className="text-xl font-semibold text-slate-900">Sign in to your account</h2>
          <form className="mt-6 space-y-4">
            <label className="block text-sm text-slate-700">
              Email
              <input
                type="email"
                placeholder="you@example.com"
                className="mt-1 w-full rounded border border-slate-200 bg-slate-100 px-3 py-2 text-sm"
              />
            </label>
            <label className="block text-sm text-slate-700">
              Password
              <input
                type="password"
                className="mt-1 w-full rounded border border-slate-200 bg-slate-100 px-3 py-2 text-sm"
              />
            </label>
            <Button type="submit" className="w-full">
              Sign In
            </Button>
          </form>
          <p className="mt-4 text-center text-sm text-slate-600">
            Don&apos;t have an account?{" "}
            <Link href="/register/" className="font-medium text-blue-600 hover:text-blue-700">
              Register
            </Link>
          </p>
        </div>
      </div>
    </div>
  );
}
