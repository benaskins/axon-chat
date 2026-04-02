import { useEffect, useRef, useState, useMemo } from "react";
import { useParams, useNavigate, useSearchParams } from "react-router";
import { Save, Trash2, Eye } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Switch } from "@/components/ui/switch";
import { Slider } from "@/components/ui/slider";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { AppHeader } from "@/components/app-header";
import { MenuButton } from "@/components/menu-button";
import { useAgent, useSaveAgent, useDeleteAgent } from "@/hooks/use-agents";
import { useModels } from "@/hooks/use-models";
import { useTools } from "@/hooks/use-tools";
import { slugify } from "@/lib/utils";

interface AgentForm {
  name: string;
  slug: string;
  avatar_emoji: string;
  tagline: string;
  system_prompt: string;
  constraints: string;
  greeting: string;
  default_model: string;
  temperature: number | null;
  think: boolean;
  top_p: number | null;
  top_k: number | null;
  min_p: number | null;
  presence_penalty: number | null;
  max_tokens: number | null;
  tools: string[];
}

const defaultForm: AgentForm = {
  name: "",
  slug: "",
  avatar_emoji: "",
  tagline: "",
  system_prompt: "",
  constraints: "",
  greeting: "",
  default_model: "",
  temperature: null,
  think: false,
  top_p: null,
  top_k: null,
  min_p: null,
  presence_penalty: null,
  max_tokens: null,
  tools: [],
};

export default function AgentEditorPage() {
  const { slug } = useParams();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const mode = slug ? "edit" : "new";
  const backHref = searchParams.get("from") || "/";

  const { data: agent } = useAgent(slug);
  const { data: models } = useModels();
  const { data: tools } = useTools();
  const saveAgent = useSaveAgent();
  const deleteAgent = useDeleteAgent();

  const [form, setForm] = useState<AgentForm>(defaultForm);
  const [showDelete, setShowDelete] = useState(false);
  const [showPreview, setShowPreview] = useState(false);
  const snapshotRef = useRef<string | null>(null);

  // Load agent data in edit mode
  useEffect(() => {
    if (agent && mode === "edit") {
      const loaded: AgentForm = {
        name: agent.name,
        slug: agent.slug,
        avatar_emoji: agent.avatar_emoji,
        tagline: agent.tagline,
        system_prompt: agent.system_prompt,
        constraints: agent.constraints,
        greeting: agent.greeting,
        default_model: agent.default_model,
        temperature: agent.temperature,
        think: agent.think ?? false,
        top_p: agent.top_p,
        top_k: agent.top_k,
        min_p: agent.min_p,
        presence_penalty: agent.presence_penalty,
        max_tokens: agent.max_tokens,
        tools: agent.tools || [],
      };
      setForm(loaded);
      snapshotRef.current = JSON.stringify(loaded);
    }
  }, [agent, mode]);

  const isDirty = useMemo(() => {
    if (!snapshotRef.current) return mode === "new" && form.name !== "";
    return JSON.stringify(form) !== snapshotRef.current;
  }, [form, mode]);

  // beforeunload warning
  useEffect(() => {
    function handler(e: BeforeUnloadEvent) {
      if (isDirty) e.preventDefault();
    }
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [isDirty]);

  // Cmd+S save
  useEffect(() => {
    function handler(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === "s") {
        e.preventDefault();
        handleSave();
      }
    }
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  });

  function updateField<K extends keyof AgentForm>(key: K, value: AgentForm[K]) {
    setForm((prev) => ({ ...prev, [key]: value }));
  }

  function toggleTool(name: string) {
    setForm((prev) => ({
      ...prev,
      tools: prev.tools.includes(name)
        ? prev.tools.filter((t) => t !== name)
        : [...prev.tools, name],
    }));
  }

  async function handleSave() {
    const agentSlug = mode === "new" ? slugify(form.name) : form.slug;
    if (!agentSlug) return;

    await saveAgent.mutateAsync({ ...form, slug: agentSlug });
    snapshotRef.current = JSON.stringify({ ...form, slug: agentSlug });

    if (mode === "new") {
      navigate(`/agents/${agentSlug}/edit?from=/`, { replace: true });
    }
  }

  async function handleDelete() {
    if (!slug) return;
    await deleteAgent.mutateAsync(slug);
    navigate("/");
  }

  const title = mode === "new" ? "New Agent" : `Edit ${form.name || slug}`;

  return (
    <div className="flex flex-col min-h-screen">
      <AppHeader
        backHref={backHref}
        title={title}
        rightContent={
          <div className="flex gap-1">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => setShowPreview(true)}
              disabled={!form.system_prompt}
            >
              <Eye className="h-4 w-4" />
            </Button>
            <Button
              variant="ghost"
              size="icon"
              onClick={handleSave}
              disabled={saveAgent.isPending}
            >
              <Save className="h-4 w-4" />
            </Button>
            {mode === "edit" && (
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setShowDelete(true)}
              >
                <Trash2 className="h-4 w-4 text-destructive" />
              </Button>
            )}
            <MenuButton />
          </div>
        }
      />

      <main className="flex-1 p-6">
        <Tabs defaultValue="persona">
          <TabsList>
            <TabsTrigger value="persona">Persona</TabsTrigger>
            <TabsTrigger value="conversation">Conversation</TabsTrigger>
            <TabsTrigger value="sampling">Sampling</TabsTrigger>
          </TabsList>

          <TabsContent value="persona" className="space-y-4 mt-4">
            <div className="grid grid-cols-[80px_1fr] gap-4">
              <div>
                <Label>Emoji</Label>
                <Input
                  value={form.avatar_emoji}
                  onChange={(e) => updateField("avatar_emoji", e.target.value)}
                  className="text-center text-2xl"
                  maxLength={4}
                />
              </div>
              <div>
                <Label>Name</Label>
                <Input
                  value={form.name}
                  onChange={(e) => updateField("name", e.target.value)}
                  placeholder="Agent name"
                />
                {mode === "new" && form.name && (
                  <p className="text-xs text-muted-foreground mt-1">
                    Slug: {slugify(form.name)}
                  </p>
                )}
              </div>
            </div>

            <div>
              <Label>Tagline</Label>
              <Input
                value={form.tagline}
                onChange={(e) => updateField("tagline", e.target.value)}
                placeholder="Brief description"
              />
            </div>

            <div>
              <Label>System Prompt</Label>
              <Textarea
                value={form.system_prompt}
                onChange={(e) => updateField("system_prompt", e.target.value)}
                placeholder="You are..."
                rows={8}
              />
            </div>

            <div>
              <Label>Constraints</Label>
              <Textarea
                value={form.constraints}
                onChange={(e) => updateField("constraints", e.target.value)}
                placeholder="Rules and boundaries"
                rows={4}
              />
            </div>

            {tools && tools.length > 0 && (
              <div>
                <Label>Tools</Label>
                <div className="grid grid-cols-2 gap-2 mt-2">
                  {tools.map((tool) => (
                    <label
                      key={tool.name}
                      className="flex items-center gap-2 text-sm cursor-pointer"
                    >
                      <Checkbox
                        checked={form.tools.includes(tool.name)}
                        onCheckedChange={() => toggleTool(tool.name)}
                      />
                      <span>{tool.name}</span>
                    </label>
                  ))}
                </div>
              </div>
            )}
          </TabsContent>

          <TabsContent value="conversation" className="space-y-4 mt-4">
            <div>
              <Label>Greeting</Label>
              <Textarea
                value={form.greeting}
                onChange={(e) => updateField("greeting", e.target.value)}
                placeholder="Hello! How can I help you today?"
                rows={3}
              />
            </div>

            <div>
              <Label>Default Model</Label>
              <select
                className="flex h-9 w-full rounded-md border bg-transparent px-3 py-1 text-sm"
                value={form.default_model}
                onChange={(e) => updateField("default_model", e.target.value)}
              >
                <option value="">Select a model</option>
                {models?.map((m) => (
                  <option key={m.Name} value={m.Name}>
                    {m.Name}
                  </option>
                ))}
              </select>
            </div>

            <div className="flex items-center gap-3">
              <Switch
                checked={form.think}
                onCheckedChange={(v) => updateField("think", v)}
              />
              <Label>Enable thinking</Label>
            </div>
          </TabsContent>

          <TabsContent value="sampling" className="space-y-6 mt-4">
            <SamplingSlider
              label="Temperature"
              value={form.temperature}
              onChange={(v) => updateField("temperature", v)}
              min={0} max={2} step={0.1}
            />
            <SamplingSlider
              label="Top P"
              value={form.top_p}
              onChange={(v) => updateField("top_p", v)}
              min={0} max={1} step={0.05}
            />
            <SamplingSlider
              label="Top K"
              value={form.top_k !== null ? form.top_k : null}
              onChange={(v) => updateField("top_k", v !== null ? Math.round(v) : null)}
              min={0} max={100} step={1}
            />
            <SamplingSlider
              label="Min P"
              value={form.min_p}
              onChange={(v) => updateField("min_p", v)}
              min={0} max={1} step={0.05}
            />
            <SamplingSlider
              label="Presence Penalty"
              value={form.presence_penalty}
              onChange={(v) => updateField("presence_penalty", v)}
              min={0} max={2} step={0.1}
            />
            <div>
              <Label>Max Tokens</Label>
              <Input
                type="number"
                value={form.max_tokens ?? ""}
                onChange={(e) =>
                  updateField("max_tokens", e.target.value ? Number(e.target.value) : null)
                }
                placeholder="Default"
              />
            </div>
          </TabsContent>
        </Tabs>
      </main>

      <AlertDialog open={showDelete} onOpenChange={setShowDelete}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete {form.name}?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete this agent and cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>Delete</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog open={showPreview} onOpenChange={setShowPreview}>
        <DialogContent className="max-w-2xl max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>System Prompt Preview</DialogTitle>
          </DialogHeader>
          <pre className="whitespace-pre-wrap text-sm font-mono bg-muted p-4 rounded">
            {form.system_prompt}
            {form.constraints && `\n\n---\n\n${form.constraints}`}
          </pre>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function SamplingSlider({
  label,
  value,
  onChange,
  min,
  max,
  step,
}: {
  label: string;
  value: number | null;
  onChange: (v: number | null) => void;
  min: number;
  max: number;
  step: number;
}) {
  const isSet = value !== null;
  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <Label>{label}</Label>
        <span className="text-sm text-muted-foreground">
          {isSet ? value : "Default"}
        </span>
      </div>
      <div className="flex items-center gap-3">
        <Slider
          value={[isSet ? value : (min + max) / 2]}
          onValueChange={(v) => {
            const val = Array.isArray(v) ? v[0] : v;
            onChange(val);
          }}
          min={min}
          max={max}
          step={step}
          className="flex-1"
        />
        {isSet && (
          <Button variant="ghost" size="sm" onClick={() => onChange(null)}>
            Reset
          </Button>
        )}
      </div>
    </div>
  );
}
