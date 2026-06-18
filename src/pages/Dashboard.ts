import { BoxRenderable, CliRenderer } from "@opentui/core";
import DashboardHeader from "../components/DashboardHeader";
import DashboardBody from "../components/DashboardBody";
import DashboardFooter from "../components/DashboardFooter";

function Dashboard(renderer: CliRenderer): BoxRenderable {
  const wrapper = new BoxRenderable(renderer, {
    id: "wrapper",
  });

  const header = DashboardHeader(renderer);
  const body = DashboardBody(renderer);
  const footer = DashboardFooter(renderer);

  // Add them to wrapper
  wrapper.add(header);
  wrapper.add(body);
  wrapper.add(footer);

  return wrapper;
}

export default Dashboard;
