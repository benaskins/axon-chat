import { Link } from "react-router";
import { Menu } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useMenu, type MenuItem } from "@/components/menu-context";
import { logout } from "@/lib/api";

export function MenuButton() {
  const { items } = useMenu();

  const defaultItems: MenuItem[] = [
    { type: "link", label: "Home", href: "/" },
    { type: "link", label: "New Agent", href: "/agents/new" },
    { type: "separator" },
    { type: "button", label: "Sign out", onClick: logout },
  ];

  const allItems =
    items.length > 0
      ? [...items, { type: "separator" as const }, ...defaultItems]
      : defaultItems;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger render={<Button variant="ghost" size="icon" />}>
        <Menu className="h-4 w-4" />
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {allItems.map((item, i) => {
          if (item.type === "separator") {
            return <DropdownMenuSeparator key={i} />;
          }
          if (item.type === "link") {
            return (
              <DropdownMenuItem key={i}>
                <Link to={item.href} className="w-full">
                  {item.label}
                </Link>
              </DropdownMenuItem>
            );
          }
          if (item.type === "button") {
            return (
              <DropdownMenuItem key={i} onClick={item.onClick}>
                {item.label}
              </DropdownMenuItem>
            );
          }
          if (item.type === "toggle") {
            return (
              <DropdownMenuItem key={i} onClick={item.onToggle}>
                {item.label}: {item.value ? "On" : "Off"}
              </DropdownMenuItem>
            );
          }
          if (item.type === "select") {
            return (
              <DropdownMenuItem key={i} className="flex-col items-start">
                <span className="text-xs text-muted-foreground">
                  {item.label}
                </span>
                <select
                  className="mt-1 text-sm bg-transparent"
                  value={item.value}
                  onChange={(e) => item.onChange(e.target.value)}
                  onClick={(e) => e.stopPropagation()}
                >
                  {item.options.map((opt) => (
                    <option key={opt.value} value={opt.value}>
                      {opt.label}
                    </option>
                  ))}
                </select>
              </DropdownMenuItem>
            );
          }
          return null;
        })}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
