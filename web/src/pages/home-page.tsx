import { Link } from "react-router";
import { Plus, Pencil } from "lucide-react";
import { Card, CardContent } from "@/components/ui/card";
import { AppHeader } from "@/components/app-header";
import { MenuButton } from "@/components/menu-button";
import { useAgents } from "@/hooks/use-agents";

export default function HomePage() {
  const { data: agents, isLoading } = useAgents();

  return (
    <div className="flex flex-col min-h-screen">
      <AppHeader title="Agents" rightContent={<MenuButton />} />
      <main className="flex-1 p-6">
        {isLoading ? (
          <p className="text-muted-foreground">Loading...</p>
        ) : (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {agents?.map((agent) => (
              <Link
                key={agent.slug}
                to={`/agents/${agent.slug}/conversations`}
                className="group"
              >
                <Card className="h-full transition-colors hover:bg-muted/50">
                  <CardContent className="relative pt-6">
                    <Link
                      to={`/agents/${agent.slug}/edit?from=/`}
                      className="absolute top-3 right-3 opacity-0 group-hover:opacity-100 transition-opacity p-1 rounded hover:bg-muted"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <Pencil className="h-3.5 w-3.5 text-muted-foreground" />
                    </Link>
                    <div className="text-3xl mb-2">{agent.avatar_emoji}</div>
                    <h3 className="font-semibold">{agent.name}</h3>
                    <p className="text-sm text-muted-foreground mt-1">
                      {agent.tagline}
                    </p>
                  </CardContent>
                </Card>
              </Link>
            ))}
            <Link to="/agents/new">
              <Card className="h-full transition-colors hover:bg-muted/50 border-dashed">
                <CardContent className="flex flex-col items-center justify-center pt-6 text-muted-foreground">
                  <Plus className="h-8 w-8 mb-2" />
                  <span className="font-medium">New Agent</span>
                </CardContent>
              </Card>
            </Link>
          </div>
        )}
      </main>
    </div>
  );
}
