import type { Metadata } from "next";
import { MessagesSquare } from "lucide-react";
import { PageHeader } from "@/components/dashboard/page-header";
import { Card } from "@/components/ui/card";
import { requireErp } from "@/features/erp/access";
import { OfflineNotice } from "@/features/erp/offline-notice";
import { load, sp1 } from "@/features/erp/shared/load";
import { ErpEmpty, ErpError } from "@/features/erp/shared/states";
import { ResetFilters, SearchFilter, SelectFilter } from "@/features/erp/shared/url-filters";
import { UrlTabs } from "@/features/erp/shared/url-tabs";
import { ConversationList, ConversationPanel } from "@/features/erp/sales/components/chat-inbox";
import { ChatbotSettingsForm, ReplyTemplates } from "@/features/erp/sales/components/chat-settings";
import { salesCaps } from "@/features/erp/sales/lib";
import { getChatbotSettings, getConversation, listConversations, listReplyTemplates } from "@/features/erp/sales/queries";
import { ownerOptions } from "@/features/erp/sales/refs";
import { CHANNEL, CONVERSATION_STATUS, opts, ss } from "@/features/erp/sales/strings";
import { getSessionUser } from "@/lib/rbac";

const s = ss.chat;
export const metadata: Metadata = { title: s.title };

export default async function ChatbotPage(props: PageProps<"/erp/sales/chatbot">) {
  const sp = await props.searchParams;
  const { access, offline } = await requireErp("sales.read");
  if (offline) return <OfflineNotice message={offline.message} />;
  const caps = salesCaps(access);
  const tab = sp1(sp.tab) ?? "inbox";
  const owners = await ownerOptions();

  let body: React.ReactNode;
  if (tab === "settings") {
    const settings = await load(getChatbotSettings);
    body = settings.data ? (
      <ChatbotSettingsForm settings={settings.data} owners={owners} admin={caps.admin} />
    ) : (
      <ErpError message={settings.error?.message} />
    );
  } else if (tab === "templates") {
    const templates = await load(listReplyTemplates);
    body = templates.data ? <ReplyTemplates templates={templates.data} canWrite={caps.write} /> : <ErpError message={templates.error?.message} />;
  } else {
    const selected = sp1(sp.c);
    const [list, detail, templates, user] = await Promise.all([
      load(() =>
        listConversations({
          status: sp1(sp.status),
          channel: sp1(sp.channel),
          mine: sp1(sp.mine) === "1" || undefined,
          q: sp1(sp.q),
          limit: 100,
        }),
      ),
      selected ? load(() => getConversation(selected)) : null,
      load(listReplyTemplates),
      getSessionUser(),
    ]);
    body = (
      <div className="grid gap-4 lg:grid-cols-[320px_1fr]">
        <Card className="flex max-h-[calc(100dvh-14rem)] min-h-[560px] flex-col gap-0 overflow-hidden py-0">
          <div className="flex flex-col gap-2 border-b p-3">
            <SearchFilter placeholder={s.searchPlaceholder} className="[&_input]:w-full" />
            <div className="flex flex-wrap gap-2">
              <SelectFilter param="status" label={s.status} allLabel={s.allStatus} options={opts(CONVERSATION_STATUS)} className="min-w-0 flex-1" />
              <SelectFilter param="channel" label={s.allChannels} allLabel={s.allChannels} options={opts(CHANNEL)} className="min-w-0 flex-1" />
              <SelectFilter param="mine" label={s.mine} allLabel={ss.all} options={[{ value: "1", label: s.mine }]} className="min-w-0 flex-1" />
              <ResetFilters params={["q", "status", "channel", "mine"]} />
            </div>
          </div>
          <div className="flex-1 overflow-y-auto">
            {list.data ? <ConversationList conversations={list.data} selected={selected} /> : <ErpError message={list.error?.message} compact />}
          </div>
        </Card>
        {detail?.data ? (
          <ConversationPanel
            key={detail.data.conversation.id}
            detail={detail.data}
            templates={templates.data ?? []}
            owners={owners}
            canWrite={caps.write}
            userId={user?.id}
          />
        ) : detail?.error ? (
          <ErpError message={detail.error.message} />
        ) : (
          <ErpEmpty icon={<MessagesSquare className="size-5" />} title={s.pick} />
        )}
      </div>
    );
  }

  return (
    <>
      <PageHeader eyebrow={ss.eyebrow} title={s.title} description={s.subtitle} />
      <UrlTabs
        label={s.title}
        tabs={[
          { value: "inbox", label: s.tabs.inbox },
          { value: "settings", label: s.tabs.settings },
          { value: "templates", label: s.tabs.templates },
        ]}
        reset={["c", "q", "status", "channel", "mine"]}
      />
      {body}
    </>
  );
}
