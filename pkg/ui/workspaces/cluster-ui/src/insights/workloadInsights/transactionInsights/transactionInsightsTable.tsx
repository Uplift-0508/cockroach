// Copyright 2022 The Cockroach Authors.
//
// Use of this software is governed by the Business Source License
// included in the file licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with
// the Business Source License, use of this software will be governed
// by the Apache License, Version 2.0, included in the file
// licenses/APL.txt.

import React from "react";
import {
  ColumnDescriptor,
  ISortedTablePagination,
  SortedTable,
  SortSetting,
} from "src/sortedtable";
import { DATE_FORMAT, Duration } from "src/util";
import { InsightExecEnum, MergedTxnInsightEvent } from "src/insights";
import {
  InsightCell,
  insightsTableTitles,
  QueriesCell,
  TransactionDetailsLink,
} from "../util";
import { Link } from "react-router-dom";
import { TimeScale } from "../../../timeScaleDropdown";

interface TransactionInsightsTable {
  data: MergedTxnInsightEvent[];
  sortSetting: SortSetting;
  onChangeSortSetting: (ss: SortSetting) => void;
  pagination: ISortedTablePagination;
  renderNoResult?: React.ReactNode;
  setTimeScale: (ts: TimeScale) => void;
}

export function makeTransactionInsightsColumns(
  setTimeScale: (ts: TimeScale) => void,
): ColumnDescriptor<MergedTxnInsightEvent>[] {
  const execType = InsightExecEnum.TRANSACTION;
  return [
    {
      name: "latestExecutionID",
      title: insightsTableTitles.latestExecutionID(execType),
      cell: item => (
        <Link to={`/insights/transaction/${item.transactionExecutionID}`}>
          {String(item.transactionExecutionID)}
        </Link>
      ),
      sort: item => item.transactionExecutionID,
    },
    {
      name: "fingerprintID",
      title: insightsTableTitles.fingerprintID(execType),
      cell: item =>
        TransactionDetailsLink(
          item.transactionFingerprintID,
          item.startTime,
          setTimeScale,
        ),
      sort: item => item.transactionFingerprintID,
    },
    {
      name: "query",
      title: insightsTableTitles.query(execType),
      cell: item => QueriesCell(item.queries, 50),
      sort: item => (item.queries?.length ? item.queries[0] : ""),
    },
    {
      name: "insights",
      title: insightsTableTitles.insights(execType),
      cell: item => item.insights.map(insight => InsightCell(insight)),
      sort: item =>
        item.insights
          ? item.insights.map(insight => insight.label).toString()
          : "",
    },
    {
      name: "startTime",
      title: insightsTableTitles.startTime(execType),
      cell: item => item.startTime?.format(DATE_FORMAT) ?? "N/A",
      sort: item => item.startTime?.unix() || 0,
    },
    {
      name: "contention",
      title: insightsTableTitles.contention(execType),
      cell: item => Duration((item.contention?.asMilliseconds() ?? 0) * 1e6),
      sort: item => item.contention?.asMilliseconds() ?? 0,
    },
    {
      name: "applicationName",
      title: insightsTableTitles.applicationName(execType),
      cell: item => item.application,
      sort: item => item.application,
    },
  ];
}

export const TransactionInsightsTable: React.FC<
  TransactionInsightsTable
> = props => {
  const columns = makeTransactionInsightsColumns(props.setTimeScale);
  return (
    <SortedTable columns={columns} className="statements-table" {...props} />
  );
};

TransactionInsightsTable.defaultProps = {};
