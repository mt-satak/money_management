import { BillResponse } from "../types";

/**
 * 指定された年月の家計簿が既に存在するかチェックする
 * @param bills 家計簿一覧
 * @param year 対象年
 * @param month 対象月
 * @returns 重複する場合はtrue、しない場合はfalse
 */
export const isDuplicateBill = (
  bills: BillResponse[],
  year: number,
  month: number,
): boolean => {
  return bills.some((bill) => bill.year === year && bill.month === month);
};

/**
 * 重複警告メッセージを生成する
 * @param year 対象年
 * @param month 対象月
 * @returns 警告メッセージ
 */
export const generateDuplicateWarningMessage = (
  year: number,
  month: number,
): string => {
  return `⚠️ ${year}年${month}月の家計簿は既に存在します`;
};

/**
 * 作成ボタンのテキストを生成する
 * @param creating 作成中かどうか
 * @param isDuplicate 重複しているかどうか
 * @returns ボタンテキスト
 */
export const generateCreateButtonText = (
  creating: boolean,
  isDuplicate: boolean,
): string => {
  if (creating) return "作成中...";
  if (isDuplicate) return "重複のため作成不可";
  return "作成";
};
