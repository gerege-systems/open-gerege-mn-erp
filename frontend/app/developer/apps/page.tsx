"use client";

import { useEffect, useState } from "react";
import { Code2, Key, Shield, Plus, CheckCircle2, Copy } from "lucide-react";
import { api } from "@/lib/api";
import { useI18n } from "@/lib/i18n";

interface OAuth2Client {
  id: string;
  client_id: string;
  client_secret: string;
  client_name: string;
  redirect_uris: string[];
  scopes: string[];
  created_at: string;
}

export default function DeveloperAppsPage() {
  const { t } = useI18n();
  const [apps, setApps] = useState<OAuth2Client[]>([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [clientName, setClientName] = useState("");
  const [redirectURIs, setRedirectURIs] = useState("http://localhost:3000/callback");
  const [copiedId, setCopiedId] = useState<string | null>(null);

  useEffect(() => {
    loadApps();
  }, []);

  const loadApps = async () => {
    try {
      const data = await api.getDeveloperApps();
      setApps(data || []);
    } catch (err) {
      console.error("Failed to load developer apps:", err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreateApp = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const uris = redirectURIs.split(",").map((s) => s.trim()).filter(Boolean);
      // erp.read/erp.write are legacy compatibility scope names kept through the
      // Gerege Nexus rebrand: the provider grants exactly these strings and
      // existing integrations request them by name.
      await api.createDeveloperApp(clientName, uris, ["openid", "profile", "erp.read", "erp.write"]);
      setClientName("");
      setShowModal(false);
      loadApps();
    } catch (err: any) {
      alert(err.message || "Failed to create OAuth2 app");
    }
  };

  const copyToClipboard = (text: string, id: string) => {
    navigator.clipboard.writeText(text);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  return (
    <div className="space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold text-slate-900 flex items-center space-x-2">
              <Code2 className="w-7 h-7 text-indigo-600" />
              <span>{t("developer.view.title")}</span>
            </h1>
            <p className="text-sm text-slate-500 mt-1">
              {t("developer.view.subtitle")}
            </p>
          </div>
          <button
            onClick={() => setShowModal(true)}
            className="bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-lg text-sm font-semibold flex items-center space-x-2 shadow-sm transition"
          >
            <Plus className="w-4 h-4" />
            <span>{t("developer.action.create")}</span>
          </button>
        </div>

        {/* OIDC Config Banner */}
        <div className="p-4 bg-slate-900 text-white rounded-xl shadow-sm flex items-center justify-between border border-slate-800">
          <div className="flex items-center space-x-3">
            <Shield className="w-8 h-8 text-cyan-400" />
            <div>
              <h3 className="font-semibold text-sm">{t("developer.field.discovery_endpoint")}</h3>
              <p className="text-xs text-slate-400 font-mono">/.well-known/openid-configuration</p>
            </div>
          </div>
          <span className="text-xs bg-emerald-500/20 text-emerald-300 font-mono px-3 py-1 rounded-full border border-emerald-500/30">
            Active SSO Provider
          </span>
        </div>

        {loading ? (
          <div className="p-12 text-center text-slate-400">{t("developer.message.loading")}</div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {apps.map((app) => (
              <div key={app.id} className="bg-white p-5 rounded-xl border border-slate-200 shadow-sm space-y-4">
                <div className="flex items-center justify-between">
                  <h3 className="font-bold text-slate-900 text-base">{app.client_name}</h3>
                  <span className="text-xs bg-indigo-50 text-indigo-700 font-medium px-2.5 py-0.5 rounded-full">
                    OAuth 2.0
                  </span>
                </div>

                <div className="space-y-2 text-xs font-mono bg-slate-50 p-3 rounded-lg border border-slate-200">
                  <div className="flex items-center justify-between">
                    <span className="text-slate-500">Client ID:</span>
                    <div className="flex items-center space-x-2">
                      <span className="text-slate-900 font-bold">{app.client_id}</span>
                      <button onClick={() => copyToClipboard(app.client_id, app.client_id)}>
                        {copiedId === app.client_id ? <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" /> : <Copy className="w-3.5 h-3.5 text-slate-400" />}
                      </button>
                    </div>
                  </div>
                  <div className="flex items-center justify-between">
                    <span className="text-slate-500">Client Secret:</span>
                    {/* The API returns the secret only in the response that
                        creates the client; later reads redact it. */}
                    {app.client_secret ? (
                      <div className="flex items-center space-x-2">
                        <span className="text-slate-900 font-bold">{app.client_secret}</span>
                        <button onClick={() => copyToClipboard(app.client_secret, `${app.client_id}-secret`)}>
                          {copiedId === `${app.client_id}-secret` ? (
                            <CheckCircle2 className="w-3.5 h-3.5 text-emerald-600" />
                          ) : (
                            <Copy className="w-3.5 h-3.5 text-slate-400" />
                          )}
                        </button>
                      </div>
                    ) : (
                      <span className="text-slate-400 italic">{t("developer.message.secret_hidden")}</span>
                    )}
                  </div>
                </div>

                <div>
                  <span className="text-xs font-semibold text-slate-700 block mb-1">{t("developer.field.redirect_uris")}</span>
                  <div className="flex flex-wrap gap-1">
                    {app.redirect_uris.map((uri, idx) => (
                      <span key={idx} className="text-xs bg-slate-100 text-slate-600 px-2 py-0.5 rounded font-mono">
                        {uri}
                      </span>
                    ))}
                  </div>
                </div>

                <div>
                  <span className="text-xs font-semibold text-slate-700 block mb-1">{t("developer.field.scopes")}</span>
                  <div className="flex flex-wrap gap-1">
                    {app.scopes.map((scope, idx) => (
                      <span key={idx} className="text-xs bg-blue-50 text-blue-700 px-2 py-0.5 rounded font-mono">
                        {scope}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Modal */}
        {showModal && (
          <div className="fixed inset-0 bg-slate-900/60 flex items-center justify-center z-50 p-4 backdrop-blur-sm">
            <div className="bg-white rounded-xl max-w-md w-full p-6 shadow-2xl space-y-4">
              <h2 className="text-lg font-bold text-slate-900">{t("developer.view.create_title")}</h2>
              <form onSubmit={handleCreateApp} className="space-y-3">
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">Application Name *</label>
                  <input
                    type="text"
                    placeholder="e.g. My External Mobile App"
                    value={clientName}
                    onChange={(e) => setClientName(e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500"
                    required
                  />
                </div>
                <div>
                  <label className="block text-xs font-semibold text-slate-700 mb-1">Redirect URIs (comma-separated) *</label>
                  <input
                    type="text"
                    value={redirectURIs}
                    onChange={(e) => setRedirectURIs(e.target.value)}
                    className="w-full px-3 py-2 text-sm border border-slate-300 rounded-lg focus:ring-2 focus:ring-indigo-500 font-mono"
                    required
                  />
                </div>
                <div className="flex justify-end space-x-2 pt-2">
                  <button
                    type="button"
                    onClick={() => setShowModal(false)}
                    className="px-4 py-2 text-sm text-slate-600 hover:bg-slate-100 rounded-lg"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="px-4 py-2 text-sm bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg font-semibold"
                  >
                    Register App
                  </button>
                </div>
              </form>
            </div>
          </div>
        )}
    </div>
  );
}
