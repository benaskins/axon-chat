import { useEffect, useState, useCallback } from "react";
import { CheckCircle, XCircle, Clock, Circle } from "lucide-react";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { BlurredImage } from "@/components/blurred-image";
import { authenticatedFetch } from "@/lib/api";
import { timeAgo } from "@/lib/utils";
import type { Task, GalleryImage } from "@/lib/types";

interface TaskDrawerProps {
  open: boolean;
  onClose: () => void;
  agentSlug: string;
}

export function TaskDrawer({ open, onClose, agentSlug }: TaskDrawerProps) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [gallery, setGallery] = useState<GalleryImage[]>([]);
  const [selectedImage, setSelectedImage] = useState<GalleryImage | null>(null);

  const loadTasks = useCallback(async () => {
    try {
      const resp = await authenticatedFetch(`/api/tasks?agent=${agentSlug}`);
      if (resp.ok) setTasks(await resp.json());
    } catch { /* ignore */ }
  }, [agentSlug]);

  const loadGallery = useCallback(async () => {
    try {
      const resp = await authenticatedFetch(`/api/agents/${agentSlug}/gallery`);
      if (resp.ok) setGallery(await resp.json());
    } catch { /* ignore */ }
  }, [agentSlug]);

  useEffect(() => {
    if (open) {
      loadTasks();
      loadGallery();
    }
  }, [open, loadTasks, loadGallery]);

  async function setBaseImage(imageId: string) {
    await authenticatedFetch(`/api/agents/${agentSlug}/gallery/${imageId}/base`, {
      method: "PUT",
    });
    loadGallery();
  }

  function statusIcon(status: string) {
    switch (status) {
      case "completed": return <CheckCircle className="h-4 w-4 text-green-500" />;
      case "failed": return <XCircle className="h-4 w-4 text-red-500" />;
      case "running": return <Clock className="h-4 w-4 text-yellow-500" />;
      default: return <Circle className="h-4 w-4 text-muted-foreground" />;
    }
  }

  return (
    <>
      <Sheet open={open} onOpenChange={(isOpen) => !isOpen && onClose()}>
        <SheetContent className="w-[400px] sm:w-[540px]">
          <SheetHeader>
            <SheetTitle>Tasks & Gallery</SheetTitle>
          </SheetHeader>
          <Tabs defaultValue="tasks" className="mt-4">
            <TabsList>
              <TabsTrigger value="tasks">Tasks</TabsTrigger>
              <TabsTrigger value="gallery">Gallery</TabsTrigger>
            </TabsList>
            <TabsContent value="tasks" className="space-y-2 mt-4">
              {tasks.length === 0 ? (
                <p className="text-sm text-muted-foreground">No tasks yet</p>
              ) : (
                tasks.map((task) => (
                  <div key={task.id} className="flex items-start gap-2 rounded border p-3">
                    {statusIcon(task.status)}
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <Badge variant="outline" className="text-xs">{task.type}</Badge>
                        <span className="text-xs text-muted-foreground">{timeAgo(task.updated_at)}</span>
                      </div>
                      <p className="text-sm mt-1">{task.description}</p>
                      {task.result_summary && (
                        <p className="text-xs text-muted-foreground mt-1">{task.result_summary}</p>
                      )}
                      {task.error && (
                        <p className="text-xs text-red-500 mt-1">{task.error}</p>
                      )}
                    </div>
                  </div>
                ))
              )}
            </TabsContent>
            <TabsContent value="gallery" className="mt-4">
              {gallery.length === 0 ? (
                <p className="text-sm text-muted-foreground">No images yet</p>
              ) : (
                <div className="grid grid-cols-2 gap-2">
                  {gallery.map((img) => (
                    <div
                      key={img.id}
                      className="relative cursor-pointer rounded overflow-hidden"
                      onClick={() => setSelectedImage(img)}
                    >
                      <BlurredImage
                        src={img.url}
                        isNsfw={img.is_nsfw}
                        className="w-full aspect-square object-cover"
                      />
                      {img.is_base && (
                        <Badge className="absolute top-1 left-1 text-xs">Base</Badge>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </TabsContent>
          </Tabs>
        </SheetContent>
      </Sheet>

      <Dialog open={!!selectedImage} onOpenChange={(isOpen) => !isOpen && setSelectedImage(null)}>
        <DialogContent className="max-w-3xl">
          {selectedImage && (
            <div className="space-y-4">
              <img src={selectedImage.url} alt="" className="w-full rounded" />
              <div className="space-y-1 text-sm">
                <p>{selectedImage.prompt}</p>
                <p className="text-muted-foreground">{selectedImage.model} · {timeAgo(selectedImage.created_at)}</p>
              </div>
              {!selectedImage.is_base && (
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setBaseImage(selectedImage.id);
                    setSelectedImage(null);
                  }}
                >
                  Set as Base Image
                </Button>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}
