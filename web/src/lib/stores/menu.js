import { writable } from 'svelte/store';

export const menuItems = writable([]);

export function setMenuItems(items) {
  menuItems.set(items);
}

export function clearMenuItems() {
  menuItems.set([]);
}
