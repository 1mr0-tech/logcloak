import {
  Accordion,
  AccordionItem,
  AccordionTrigger,
  AccordionContent,
} from '@/components/animate-ui/components/radix/accordion';

export default function RbacFaq() {
  return (
    <Accordion type="single" collapsible className="rounded-lg border border-border">
      <AccordionItem value="rbac" className="border-b-0 px-4">
        <AccordionTrigger className="py-3 text-sm font-semibold text-foreground">
          RBAC for platform teams
        </AccordionTrigger>
        <AccordionContent className="pb-4 text-sm text-muted-foreground">
          <p className="mb-3">
            Two ClusterRoles ship with the chart for managing <code className="rounded border border-border bg-white/5 px-1 py-0.5 font-mono text-xs">MaskingPolicy</code> resources
            without cluster-admin: <code className="rounded border border-border bg-white/5 px-1 py-0.5 font-mono text-xs">logcloak-admin</code> (full management) and
            <code className="rounded border border-border bg-white/5 px-1 py-0.5 font-mono text-xs"> logcloak-viewer</code> (read-only).
          </p>
          <pre className="overflow-x-auto rounded-lg border border-border bg-[#0d0d0f] p-4 font-mono text-[0.8rem] leading-relaxed text-foreground/90">
{`kubectl create clusterrolebinding ops-admin \\
  --clusterrole=logcloak-admin --user=ops@company.com`}
          </pre>
        </AccordionContent>
      </AccordionItem>
    </Accordion>
  );
}
