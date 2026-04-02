import { useEffect, useState } from "react";
import { useParams, useNavigate, Link } from "react-router";
import { Plus, Trash2, MessageSquare } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { AppHeader } from "@/components/app-header";
import { MenuButton } from "@/components/menu-button";
import { useMenu } from "@/components/menu-context";
import { useAgent } from "@/hooks/use-agents";
import {
  useConversations,
  useCreateConversation,
  useDeleteConversation,
} from "@/hooks/use-conversations";
import { timeAgo } from "@/lib/utils";

export default function ConversationsPage() {
  const { slug } = useParams();
  const navigate = useNavigate();
  const { setItems, clearItems } = useMenu();
  const { data: agent } = useAgent(slug);
  const { data: conversations, isLoading } = useConversations(slug);
  const createConversation = useCreateConversation();
  const deleteConversation = useDeleteConversation();
  const [deletingId, setDeletingId] = useState<string | null>(null);

  useEffect(() => {
    if (agent) {
      setItems([
        { type: "link", label: "Edit Agent", href: `/agents/${slug}/edit?from=/agents/${slug}/conversations` },
      ]);
    }
    return clearItems;
  }, [agent, slug, setItems, clearItems]);

  async function handleNewConversation() {
    if (!slug) return;
    const conversation = await createConversation.mutateAsync(slug);
    navigate(`/chat/${slug}/${conversation.id}`);
  }

  async function handleDelete(id: string) {
    await deleteConversation.mutateAsync(id);
    setDeletingId(null);
  }

  const title = agent
    ? `${agent.avatar_emoji} ${agent.name}`
    : "Conversations";

  return (
    <div className="flex flex-col min-h-screen">
      <AppHeader
        backHref="/"
        title={title}
        rightContent={
          <div className="flex gap-1">
            <Button variant="ghost" size="icon" onClick={handleNewConversation}>
              <Plus className="h-4 w-4" />
            </Button>
            <MenuButton />
          </div>
        }
      />
      <main className="flex-1 p-6">
        {isLoading ? (
          <p className="text-muted-foreground">Loading...</p>
        ) : conversations?.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
            <MessageSquare className="h-12 w-12 mb-4" />
            <p>No conversations yet</p>
            <Button variant="outline" className="mt-4" onClick={handleNewConversation}>
              Start a conversation
            </Button>
          </div>
        ) : (
          <div className="space-y-2">
            {conversations?.map((conv) => (
              <div
                key={conv.id}
                className="flex items-center gap-3 rounded-lg border p-4 hover:bg-muted/50 transition-colors"
              >
                <Link
                  to={`/chat/${slug}/${conv.id}`}
                  className="flex-1 min-w-0"
                >
                  <div className="font-medium truncate">
                    {conv.title || "Untitled"}
                  </div>
                  <div className="text-sm text-muted-foreground">
                    {conv.message_count} messages · {timeAgo(conv.updated_at)}
                  </div>
                </Link>
                <AlertDialog
                  open={deletingId === conv.id}
                  onOpenChange={(open) => !open && setDeletingId(null)}
                >
                  <AlertDialogTrigger
                    render={
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setDeletingId(conv.id)}
                      />
                    }
                  >
                    <Trash2 className="h-4 w-4 text-muted-foreground" />
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Delete conversation?</AlertDialogTitle>
                      <AlertDialogDescription>
                        This will permanently delete this conversation and all
                        its messages.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Cancel</AlertDialogCancel>
                      <AlertDialogAction onClick={() => handleDelete(conv.id)}>
                        Delete
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            ))}
          </div>
        )}
      </main>
    </div>
  );
}
