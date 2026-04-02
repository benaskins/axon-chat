import { createBrowserRouter } from "react-router";
import AuthLayout from "@/layouts/auth-layout";
import HomePage from "@/pages/home-page";
import ConversationsPage from "@/pages/conversations-page";
import ChatPage from "@/pages/chat-page";
import AgentEditorPage from "@/pages/agent-editor-page";

export const router = createBrowserRouter([
  {
    element: <AuthLayout />,
    children: [
      { path: "/", element: <HomePage /> },
      { path: "/agents/:slug/conversations", element: <ConversationsPage /> },
      { path: "/agents/:slug/edit", element: <AgentEditorPage /> },
      { path: "/agents/new", element: <AgentEditorPage /> },
      { path: "/chat/:slug/:id", element: <ChatPage /> },
    ],
  },
]);
