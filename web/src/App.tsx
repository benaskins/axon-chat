import { Button } from "@/components/ui/button"

function App() {
  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="text-center space-y-4">
        <h1 className="text-2xl font-bold">axon-chat</h1>
        <p className="text-muted-foreground">React + shadcn/ui + Tailwind</p>
        <Button>Hello</Button>
      </div>
    </div>
  )
}

export default App
