import { setupServer } from "msw/node";
import { handlers } from "./handlers";

// MSWテスト用サーバーを設定
export const server = setupServer(...handlers);
