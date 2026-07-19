import { useQuery } from "@tanstack/react-query";
import { listNews } from "@/api/news";

// GET /api/news is live today. Query key "news" matches what
// WebSocketProvider writes into on `news_published` so live pushes merge
// straight into this cache without a separate code path.
export function useNews(limit = 15) {
  return useQuery({
    queryKey: ["news"],
    queryFn: () => listNews({ limit }),
  });
}
