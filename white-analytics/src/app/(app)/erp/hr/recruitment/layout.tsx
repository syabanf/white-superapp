import { RecruitmentNav } from "@/features/erp/recruitment/components/ui";

export default function RecruitmentLayout({ children }: LayoutProps<"/erp/hr/recruitment">) {
  return (
    <>
      <RecruitmentNav />
      {children}
    </>
  );
}
