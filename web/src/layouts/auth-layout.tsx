import { useEffect, useState } from "react";
import { Outlet } from "react-router";
import { checkAuth, redirectToLogin } from "@/lib/api";
import { MenuProvider } from "@/components/menu-context";

export default function AuthLayout() {
  const [ready, setReady] = useState(false);

  useEffect(() => {
    checkAuth().then((ok) => {
      if (ok) {
        setReady(true);
      } else {
        redirectToLogin();
      }
    });
  }, []);

  if (!ready) return null;

  return (
    <MenuProvider>
      <Outlet />
    </MenuProvider>
  );
}
