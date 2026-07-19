import { useQuery } from "@tanstack/react-query";
import { listNews } from "@/api/news";

// Deliberately a separate query key from useNews()/the dashboard panel
// (which fetch a different limit at offset 0) — sharing a key across
// different limits would let one page's cached shape leak into the other.
// This means live news_published WS pushes (which only patch ["news"])
// won't reach this page's first screen; acceptable since a paginated feed
// is read as a point-in-time list anyway.
export function useNewsPage(offset: number, limit = 20) {
  return useQuery({
    queryKey: ["newsPage", offset, limit],
    queryFn: () => listNews({ limit, offset }),
  });
}
