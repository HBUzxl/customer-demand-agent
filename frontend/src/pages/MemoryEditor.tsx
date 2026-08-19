import { useState } from "react";
import { Navigate, useNavigate, useSearchParams } from "react-router-dom";
import { memoryUpsert } from "../api/client";
import { useAuth } from "../auth/AuthProvider";
import MemoryForm, { fromValues, type MemoryValues } from "../components/MemoryForm";
import { ROLE_ADMIN, ROLE_OWNER } from "../types";

// /memory/new —— 新建记忆
export default function MemoryEditor() {
  const navigate = useNavigate();
  const { user } = useAuth();
  const canManageMemory = !!user?.roles.some((r) => r === ROLE_OWNER || r === ROLE_ADMIN);
  const [params] = useSearchParams(); // 无结果 CTA 预填（type+title）
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  if (!canManageMemory) return <Navigate to="/memory" replace />;

  async function onSubmit(v: MemoryValues) {
    setSubmitting(true);
    setError("");
    try {
      const entry = fromValues(v);
      await memoryUpsert(entry);
      navigate(`/memory/${entry.type}/${encodeURIComponent(entry.title)}`, { replace: true });
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="page">
      <div className="page-head">
        <h2>新建记忆</h2>
        <span className="crumb">NEW ENTRY</span>
      </div>
      {error && <div className="error">! {error}</div>}
      <MemoryForm
        initial={{
          type: ["threat", "compliance", "industry", "customer"].includes(params.get("type") || "")
            ? params.get("type")!
            : "customer",
          title: params.get("title") || "",
          summary: "",
          content: "",
          tags: "",
          aliases: "",
        }}
        submitting={submitting}
        onSubmit={onSubmit}
        onCancel={() => navigate("/memory")}
      />
    </div>
  );
}
