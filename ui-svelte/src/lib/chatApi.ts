import type { ChatMessage, ChatCompletionRequest, ContentPart } from "./types";

export interface StreamChunk {
  content: string;
  reasoning_content?: string;
  done: boolean;
}

export interface ChatOptions {
  temperature?: number;
}

type ChatRequestMessage = Pick<ChatMessage, "role" | "content">;

function normalizeDeltaText(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }

  if (!Array.isArray(value)) {
    return "";
  }

  return value
    .map((part) => {
      if (typeof part === "string") return part;
      if (!part || typeof part !== "object") return "";
      const text = (part as { text?: unknown }).text;
      return typeof text === "string" ? text : "";
    })
    .join("");
}

function parseSSELine(line: string): StreamChunk | null {
  const trimmed = line.trim();
  if (!trimmed || !trimmed.startsWith("data:")) {
    return null;
  }

  const data = trimmed.slice(5).trim();
  if (!data) {
    return null;
  }

  if (data === "[DONE]") {
    return { content: "", done: true };
  }

  try {
    const parsed = JSON.parse(data);
    const choice = parsed.choices?.[0];
    const delta = choice?.delta ?? choice?.message ?? {};
    const content = normalizeDeltaText(delta?.content);
    const reasoning_content = normalizeDeltaText(delta?.reasoning_content);
    const finishReason = typeof choice?.finish_reason === "string" ? choice.finish_reason.trim() : "";

    if (content || reasoning_content) {
      return { content, reasoning_content, done: false };
    }
    if (finishReason) {
      return { content: "", done: true };
    }
    return null;
  } catch {
    return null;
  }
}

function sanitizeContent(content: unknown): string | ContentPart[] | null {
  if (typeof content === "string") {
    return content;
  }

  if (!Array.isArray(content)) {
    return null;
  }

  const cleaned: ContentPart[] = [];
  for (const rawPart of content) {
    if (!rawPart || typeof rawPart !== "object") {
      continue;
    }

    const part = rawPart as Partial<ContentPart>;
    if (part.type === "text") {
      const text = (part as { text?: unknown }).text;
      if (typeof text === "string" && text.length > 0) {
        cleaned.push({ type: "text", text });
      }
      continue;
    }

    if (part.type === "image_url") {
      const imageUrl = (part as { image_url?: { url?: unknown } }).image_url;
      const url = imageUrl?.url;
      if (typeof url === "string" && url.trim() !== "") {
        cleaned.push({ type: "image_url", image_url: { url } });
      }
    }
  }

  return cleaned.length > 0 ? cleaned : null;
}

function normalizeMessages(messages: ChatMessage[]): ChatRequestMessage[] {
  const out: ChatRequestMessage[] = [];

  for (const msg of messages) {
    if (!msg || (msg.role !== "user" && msg.role !== "assistant" && msg.role !== "system")) {
      continue;
    }

    const content = sanitizeContent((msg as { content?: unknown }).content);
    if (content === null) {
      continue;
    }

    if (typeof content === "string" && content.trim() === "") {
      // Avoid sending empty assistant/system/user entries that can break some backends (TRT harmony).
      continue;
    }

    out.push({ role: msg.role, content });
  }

  return out;
}

export async function* streamChatCompletion(
  model: string,
  messages: ChatMessage[],
  signal?: AbortSignal,
  options?: ChatOptions
): AsyncGenerator<StreamChunk> {
  const request: ChatCompletionRequest = {
    model,
    messages: normalizeMessages(messages) as ChatMessage[],
    stream: true,
    temperature: options?.temperature,
  };

  const response = await fetch("/v1/chat/completions", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(request),
    signal,
  });

  if (!response.ok) {
    const errorText = await response.text();
    throw new Error(`Chat API error: ${response.status} - ${errorText}`);
  }

  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error("Response body is not readable");
  }

  const decoder = new TextDecoder();
  let buffer = "";

  try {
    while (true) {
      const { done, value } = await reader.read();

      if (done) {
        break;
      }

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split("\n");
      buffer = lines.pop() || "";

      for (const line of lines) {
        const result = parseSSELine(line);
        if (result?.done) {
          yield result;
          return;
        }
        if (result) {
          yield result;
        }
      }
    }

    // Process any remaining buffer
    const result = parseSSELine(buffer);
    if (result && !result.done) {
      yield result;
    }

    yield { content: "", done: true };
  } finally {
    reader.releaseLock();
  }
}
