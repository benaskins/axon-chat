import { createContext, useContext, useState, useCallback, type ReactNode } from "react";

export type MenuItem =
  | { type: "link"; label: string; href: string }
  | { type: "button"; label: string; onClick: () => void }
  | { type: "toggle"; label: string; value: boolean; onToggle: () => void }
  | {
      type: "select";
      label: string;
      value: string;
      options: { label: string; value: string }[];
      onChange: (value: string) => void;
    }
  | { type: "separator" };

interface MenuContextValue {
  items: MenuItem[];
  setItems: (items: MenuItem[]) => void;
  clearItems: () => void;
}

const MenuContext = createContext<MenuContextValue>({
  items: [],
  setItems: () => {},
  clearItems: () => {},
});

export function MenuProvider({ children }: { children: ReactNode }) {
  const [items, setItemsState] = useState<MenuItem[]>([]);
  const setItems = useCallback((items: MenuItem[]) => setItemsState(items), []);
  const clearItems = useCallback(() => setItemsState([]), []);

  return (
    <MenuContext.Provider value={{ items, setItems, clearItems }}>
      {children}
    </MenuContext.Provider>
  );
}

export function useMenu() {
  return useContext(MenuContext);
}
