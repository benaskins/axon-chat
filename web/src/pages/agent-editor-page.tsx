import { useParams } from "react-router";

export default function AgentEditorPage() {
  const { slug } = useParams();
  const mode = slug ? "edit" : "new";
  return (
    <div className="p-8">
      <h1 className="text-2xl font-bold">
        {mode === "new" ? "New Agent" : `Edit ${slug}`}
      </h1>
    </div>
  );
}
