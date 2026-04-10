// @ts-nocheck
"use client";

import { useRef, useState } from "react";

const ALL_SOURCES = ["JobOKO", "JobsGO", "TopDev", "TopCV", "ITviec", "Glints"];
const ALL_LEVELS = [
  { key: "intern", label: "Intern" },
  { key: "fresher", label: "Fresher" },
  { key: "junior", label: "Junior" },
  { key: "middle", label: "Middle" },
  { key: "senior", label: "Senior" },
  { key: "manager", label: "Manager" },
];
const ALL_LOCATIONS = [
  { key: "tphcm", label: "TP.HCM" },
  { key: "hanoi", label: "Hà Nội" },
  { key: "danang", label: "Đà Nẵng" },
  { key: "remote", label: "Remote" },
];

export default function Home() {
  const resultsRef = useRef(null);
  const [keyword, setKeyword] = useState("");
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [stats, setStats] = useState({ total: 0, bySource: {} });
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);

  const [selectedSources, setSelectedSources] = useState([...ALL_SOURCES]);
  const [selectedLevels, setSelectedLevels] = useState([]);
  const [selectedLocations, setSelectedLocations] = useState([]);

  const toggleSelection = (setter, state, value) => {
    if (state.includes(value)) {
      setter(state.filter(v => v !== value));
    } else {
      setter([...state, value]);
    }
  };

  const formatExperience = (raw) => {
    const text = String(raw || "").trim();
    if (!text || text.toLowerCase() === "unknown") return "";

    const range = text.match(/(\d+)\s*[-~]\s*(\d+)\s*(?:\+)?\s*(?:n\u0103m|nam|years?)/i);
    if (range) return `${range[1]}-${range[2]} n\u0103m`;

    const single = text.match(/(\d+)\s*(\+)?\s*(?:n\u0103m|nam|years?)/i);
    if (single) return single[2] ? `${single[1]}+ n\u0103m` : `${single[1]} n\u0103m`;

    const onlyNumber = text.match(/^(\d+)$/);
    if (onlyNumber) return `${onlyNumber[1]} n\u0103m`;

    return text;
  };

  const formatLevel = (raw) => {
    const level = String(raw || "").trim().toLowerCase();
    if (!level || level === "unknown") return "";

    const labels = {
      intern: "Intern",
      fresher: "Fresher",
      junior: "Junior",
      middle: "Middle",
      senior: "Senior",
      manager: "Manager",
    };
    return labels[level] || level[0].toUpperCase() + level.slice(1);
  };

  const fetchJobs = async (targetPage = 1) => {
    if (!keyword.trim()) return;

    setLoading(true);
    setError("");
    setJobs([]);
    
    try {
      const params = new URLSearchParams();
      params.append("keyword", keyword);
      params.append("page", targetPage);
      selectedLevels.forEach(l => params.append("levels", l));
      selectedLocations.forEach(l => params.append("locations", l));
      selectedSources.forEach(s => params.append("sources", s));

      const res = await fetch(`http://localhost:8080/api/jobs/search?${params.toString()}`);
      
      if (!res.ok) {
        throw new Error("Lỗi khi fetch dữ liệu từ backend");
      }
      
      const data = await res.json();
      setJobs(data.jobs || []);
      setStats({
        total: data.totalCount || 0,
        bySource: data.countBySource || {}
      });
      setPage(data.filter?.Page || targetPage);
      setTotalPages(data.totalPages || 1);

      // Tự động cuộn về phần kết quả (không nhảy lên đầu trang).
      requestAnimationFrame(() => {
        resultsRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
      });

    } catch (err) {
      console.error(err);
      setError("Không thể kết nối đến máy chủ. Hãy đảm bảo backend Go đang chạy trên cổng 8080.");
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = (e) => {
    if (e) e.preventDefault();
    fetchJobs(1);
  };

  return (
    <>
      <header className="header">
        <div className="container header-content">
          <div className="logo">
            <div className="logo-icon">💼</div>
            JobAggregator
          </div>
        </div>
      </header>

      <main className="main-content">
        <section className="hero">
          <div className="container">
            <h1><span className="gradient-text">Tìm Việc IT</span> Đa Nguồn</h1>
            <p>Next.js Frontend Client + Go Backend Crawler. Tìm kiếm siêu tốc từ nhiều nền tảng.</p>
            
            <form onSubmit={handleSearch} className="search-box">
              <input 
                type="text" 
                className="search-input" 
                placeholder="Nhập vị trí: nodejs, react, golang..."
                value={keyword}
                onChange={(e) => setKeyword(e.target.value)}
                required
              />
              <button type="submit" className="search-btn" disabled={loading}>
                {loading ? "Đang tìm..." : "Tìm kiếm"}
              </button>
            </form>

            <div className="filters-wrapper" style={{ textAlign: "left" }}>
              <div className="filter-group">
                <label>📍 Địa điểm (Mặc định: Tất cả)</label>
                <div className="chip-list">
                  {ALL_LOCATIONS.map(loc => (
                    <div 
                      key={loc.key} 
                      className={`chip ${selectedLocations.includes(loc.key) ? 'active' : ''}`}
                      onClick={() => toggleSelection(setSelectedLocations, selectedLocations, loc.key)}
                    >
                      {loc.label}
                    </div>
                  ))}
                </div>
              </div>

              <div className="filter-group">
                <label>📈 Trình độ (Mặc định: Tất cả)</label>
                <div className="chip-list">
                  {ALL_LEVELS.map(lvl => (
                    <div 
                      key={lvl.key} 
                      className={`chip ${selectedLevels.includes(lvl.key) ? 'active' : ''}`}
                      onClick={() => toggleSelection(setSelectedLevels, selectedLevels, lvl.key)}
                    >
                      {lvl.label}
                    </div>
                  ))}
                </div>
              </div>

              <div className="filter-group">
                <label>🌐 Nguồn cào dữ liệu</label>
                <div className="chip-list">
                  {ALL_SOURCES.map(src => (
                    <div 
                      key={src} 
                      className={`chip ${selectedSources.includes(src) ? 'active' : ''}`}
                      onClick={() => toggleSelection(setSelectedSources, selectedSources, src)}
                    >
                      {src}
                    </div>
                  ))}
                </div>
              </div>
            </div>

          </div>
        </section>

        <section className="container results-section" ref={resultsRef}>
          {loading && (
            <div className="loader-container">
              <div className="spinner"></div>
              <p>Đang tìm kiếm tự động từ các nguồn...</p>
            </div>
          )}

          {error && (
            <div style={{ color: "#ff5252", textAlign: "center", padding: "20px", background: "rgba(255,0,0,0.1)", borderRadius: "12px", border: "1px solid rgba(255,0,0,0.3)", marginBottom: "40px" }}>
              {error}
            </div>
          )}

          {!loading && !error && jobs.length > 0 && (
            <>
              <div className="job-stats">
                <div>
                  Tìm thấy <strong style={{ color: "white" }}>{stats.total}</strong> việc làm
                </div>
                <div style={{ display: "flex", gap: "10px", flexWrap: "wrap", justifyContent: "flex-end" }}>
                   {Object.entries(stats.bySource).map(([src, count]) => (
                     <span key={src} style={{ fontSize: "0.8rem", background: "rgba(255,255,255,0.1)", padding: "4px 10px", borderRadius: "12px" }}>
                        {src}: {count}
                     </span>
                   ))}
                </div>
              </div>

              <div className="job-grid">
                {jobs.map((job, idx) => (
                  <div key={idx} className="job-card">
                    <div className="job-header">
                      <h3 className="job-title" title={job.title}>{job.title}</h3>
                      <span className="job-source">{job.source}</span>
                    </div>
                    
                    <div className="job-company">{job.companyName}</div>
                    
                    <div className="job-meta">
                      {job.location && (
                        <div className="meta-item">
                          <span className="meta-icon">📍</span>
                          {job.location}
                        </div>
                      )}
                      
                      {job.salary && (
                        <div className="meta-item">
                          <span className="meta-icon">💰</span>
                          {job.salary}
                        </div>
                      )}
                      
                      {job.source === "ITviec" ? (
                        formatLevel(job.level) && (
                          <div className="meta-item">
                            <span className="meta-icon">📈</span>
                            {formatLevel(job.level)}
                          </div>
                        )
                      ) : (
                        formatExperience(job.experience) && (
                          <div className="meta-item">
                            <span className="meta-icon">📈</span>
                            {formatExperience(job.experience)}
                          </div>
                        )
                      )}
                    </div>
                    
                    <div className="job-footer">
                      <span style={{ fontSize: "0.8rem", color: "var(--text-muted)" }}>
                        {job.postedDate ? new Date(job.postedDate).toLocaleDateString("vi-VN") : "Gần đây"}
                      </span>
                      <a href={job.url} target="_blank" rel="noreferrer" className="btn-apply">
                        Ứng tuyển
                      </a>
                    </div>
                  </div>
                ))}
              </div>

              {totalPages > 1 && (
                <div className="pagination" style={{ display: "flex", justifyContent: "center", alignItems: "center", gap: "10px", marginTop: "40px" }}>
                  <button 
                    onClick={() => fetchJobs(page - 1)} 
                    disabled={page <= 1 || loading}
                    style={{ padding: "8px 16px", background: "var(--surface)", color: "white", border: "1px solid var(--border)", borderRadius: "8px", cursor: (page <= 1 || loading) ? "not-allowed" : "pointer" }}
                  >
                    Trước
                  </button>
                  <span style={{ padding: "8px 16px", color: "var(--text-muted)" }}>Trang {page} / {totalPages}</span>
                  <button 
                    onClick={() => fetchJobs(page + 1)} 
                    disabled={page >= totalPages || loading}
                    style={{ padding: "8px 16px", background: "var(--surface)", color: "white", border: "1px solid var(--border)", borderRadius: "8px", cursor: (page >= totalPages || loading) ? "not-allowed" : "pointer" }}
                  >
                    Sau
                  </button>
                </div>
              )}
            </>
          )}

          {!loading && !error && jobs.length === 0 && keyword && (
             <div style={{ textAlign: "center", padding: "60px 0", color: "var(--text-muted)" }}>
                <h2>Không tìm thấy việc làm nào</h2>
                <p>Bạn hãy thử kiểm tra lại từ khóa hoặc giảm bớt bộ lọc nhé.</p>
             </div>
          )}
        </section>
      </main>
    </>
  );
}

