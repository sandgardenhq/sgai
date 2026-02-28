import { ChatButton } from "./ChatButton";
import { ChatDrawer } from "./ChatDrawer";

export function ChatAssistant() {
  return (
    <>
      <ChatButton />
      <ChatDrawer />
    </>
  );
}

export { ChatButton } from "./ChatButton";
export { ChatDrawer } from "./ChatDrawer";
export { ChatInput } from "./ChatInput";
export { ChatMessage } from "./ChatMessage";
