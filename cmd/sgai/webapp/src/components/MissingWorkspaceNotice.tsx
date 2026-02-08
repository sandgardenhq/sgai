import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { ArrowLeft } from "lucide-react";
import { useNavigate } from "react-router";

interface MissingWorkspaceNoticeProps {
  title?: string;
  description?: string;
}

export function MissingWorkspaceNotice({
  title = "Workspace required",
  description = "Open the GOAL composer from a workspace to continue.",
}: MissingWorkspaceNoticeProps) {
  const navigate = useNavigate();

  return (
    <div className="max-w-2xl mx-auto py-8">
      <Alert className="mb-4">
        <AlertTitle>{title}</AlertTitle>
        <AlertDescription>{description}</AlertDescription>
      </Alert>
      <Button variant="outline" onClick={() => navigate("/")}>
        <ArrowLeft className="mr-2 h-4 w-4" />
        Back to Dashboard
      </Button>
    </div>
  );
}
