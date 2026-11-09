import type { ReactNode } from "react";
import { Header } from "@/components/shop/header";
import { Footer } from "@/components/shop/footer";

export default function ShopLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <Header />
      <main className="mx-auto min-h-[70vh] max-w-7xl px-4 py-8 sm:px-6">{children}</main>
      <Footer />
    </>
  );
}
