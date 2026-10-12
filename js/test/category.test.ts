import { describe, it, expect } from "vitest";
import { readFileSync } from "node:fs";
import { assembleNtcharts } from "../src/ntcharts/index.js";
import { expectGolden } from "./helpers.js";

const input = (name: string): any => JSON.parse(readFileSync(new URL(`../../testdata/fixtures-terminal/${name}.json`,import.meta.url),"utf8"));

describe("categorical X axes", () => {
 for (const name of ["line-categorical","scatter-categorical","slope","strip","strip-dense"]) it(`compiles ${name}`, () => {
  const out=assembleNtcharts(input(name));
  expect(out.x_axis?.type).toBe("category");
  expect(out.data.series.every(s=>s.values?.every(p=>typeof p.x==="string"&&Number.isFinite(p.y)))).toBe(true);
  expectGolden(name,out,"terminal");
 });
 it("retains missing categories independently of line points",()=>{
  const out=assembleNtcharts(input("line-categorical"));
  expect(out.x_axis?.labels).toEqual(["Start","Gap","Finish"]);
  expect(out.data.series[0].values?.map(p=>p.x)).toEqual(["Start","Finish"]);
  expect(out._warnings?.[0].code).toBe("invalid-category-point");
 });
 it("keeps repeated scatter observations",()=>{
  const out=assembleNtcharts(input("scatter-categorical"));
  expect(out.x_axis?.labels).toEqual(["B","A","C"]);
  expect(out.data.series[0].values).toHaveLength(4);
 });
 it("sorts categorical line points by semantic category order",()=>{
  const doc=input("line-categorical");
  doc.data.values=[{stage:"Mar",score:3},{stage:"Jan",score:1},{stage:"Feb",score:2}];
  doc.semantic_types={stage:{semanticType:"Month"}};
  const out=assembleNtcharts(doc);
  expect(out.x_axis?.labels).toEqual(["Jan","Feb","Mar"]);
  expect(out.data.series[0].values?.map(p=>p.x)).toEqual(["Jan","Feb","Mar"]);
 });
 it("naturally orders the two slope periods",()=>{
  const out=assembleNtcharts(input("slope"));
  expect(out.x_axis?.labels).toEqual(["2020","2025"]);
  expect(out.data.series).toHaveLength(2);
  expect(out._warnings).toBeUndefined();
 });
 it("keeps detail groups separate without a legend",()=>{
  const doc=input("slope");doc.chart_spec.encodings.detail=doc.chart_spec.encodings.color;delete doc.chart_spec.encodings.color;
  const out=assembleNtcharts(doc);
  expect(out.data.series).toHaveLength(2);expect(out.options?.show_legend).toBe(false);
 });
 it("reports omitted Strip jitter",()=>{
  const doc=input("strip");delete doc.chart_spec.chartProperties;
  expect(assembleNtcharts(doc)._warnings?.[0].code).toBe("strip-jitter-unsupported");
 });
 it("warns and stays linear on categorical X log requests",()=>{
  const doc=input("scatter-categorical");doc.chart_spec.chartProperties={logScale_x:true,logScale_y:true};
  const out=assembleNtcharts(doc);
  expect(out.x_axis?.scale).toBeUndefined();expect(out.y_axis?.scale).toBe("log");
  expect(out._warnings?.[0].message).toContain("categorical");
 });
 it("separates flattened facets rather than connecting duplicate periods",()=>{
  const doc=input("slope");doc.data.values.forEach((r:any)=>r.region="East");
  doc.data.values.push(...doc.data.values.map((r:any)=>({...r,region:"West"})));
  doc.chart_spec.encodings.column={field:"region"};
  const out=assembleNtcharts(doc);
  expect(out.data.series).toHaveLength(4);expect(out._warnings?.[0].code).toBe("facets-flattened");
 });
 it("rejects duplicate line observations without a reduction",()=>{
  const doc=input("slope");doc.data.values.push({...doc.data.values[0]});
  expect(()=>assembleNtcharts(doc)).toThrow(/duplicate category/);
 });
});
