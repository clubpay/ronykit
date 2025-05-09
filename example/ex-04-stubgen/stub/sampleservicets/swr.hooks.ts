"use client";

// Code generated by RonyKIT Stub Generator (TypeScript); DO NOT EDIT.
// @ts-ignore
import useSWR, { SWRConfiguration } from "swr";
import {
	sampleServiceStub,
	ErrorMessage,
	KeyValue,
	Location,
	SimpleHdr,
	Time,
	VeryComplexRequest,
	VeryComplexResponse,
	zone,
	zoneTrans,
} from "./stub";

export function useComplexDummy(
	stub: sampleServiceStub,
	req: VeryComplexRequest,
	reqHeader?: HeadersInit,
	options?: Partial<SWRConfiguration<VeryComplexResponse>>,
) {
	return useSWR(
		[req, "ComplexDummy"],
		(req) => {
			return stub.complexDummy(req[0], reqHeader);
		},
		options,
	);
}
export function useComplexDummy2(
	stub: sampleServiceStub,
	req: VeryComplexRequest,
	reqHeader?: HeadersInit,
	options?: Partial<SWRConfiguration<VeryComplexResponse>>,
) {
	return useSWR(
		[req, "ComplexDummy2"],
		(req) => {
			return stub.complexDummy2(req[0], reqHeader);
		},
		options,
	);
}
export function useGetComplexDummy(
	stub: sampleServiceStub,
	req: VeryComplexRequest,
	reqHeader?: HeadersInit,
	options?: Partial<SWRConfiguration<VeryComplexResponse>>,
) {
	return useSWR(
		[req, "GetComplexDummy"],
		(req) => {
			return stub.getComplexDummy(req[0], reqHeader);
		},
		options,
	);
}
