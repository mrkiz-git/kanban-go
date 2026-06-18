"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";

export default function AdminIndexPage() {
  const router = useRouter();

  useEffect(() => {
    router.replace("/admin/users/");
  }, [router]);

  return (
    <div className="flex flex-1 items-center justify-center text-sm text-slate-600">
      Redirecting to admin users…
    </div>
  );
}
