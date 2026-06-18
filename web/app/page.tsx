"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

export default function HomePage() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/boards/");
  }, [router]);

  return (
    <div className="flex min-h-screen items-center justify-center text-sm text-slate-600">
      Redirecting to boards…
    </div>
  );
}
