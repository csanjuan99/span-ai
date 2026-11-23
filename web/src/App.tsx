import { BrowserRouter, Route, Routes } from "react-router"

import { Desk } from "@/pages/Desk"
import { SpanPage } from "@/pages/SpanPage"

/** Desk at the root, one span page for a run. */
export function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<Desk />} />
        <Route path="/spans/:id" element={<SpanPage />} />
      </Routes>
    </BrowserRouter>
  )
}
