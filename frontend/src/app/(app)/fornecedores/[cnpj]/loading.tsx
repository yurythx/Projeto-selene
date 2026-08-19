import { Skeleton } from "@/components/ui/skeleton";

export default function FornecedorDetalheLoading() {
  return (
    <div className="max-w-3xl space-y-6">
      <div>
        <Skeleton className="h-8 w-72" />
        <Skeleton className="mt-2 h-4 w-40" />
      </div>
      <Skeleton className="h-40 w-full" />
      <Skeleton className="h-40 w-full" />
    </div>
  );
}
