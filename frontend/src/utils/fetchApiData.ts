import axios from 'axios';
import { Feature, Geometry } from 'geojson';

const API_BASE_URL = process.env.REACT_APP_API_BASE_URL;
if (!API_BASE_URL) {
    throw new Error('REACT_APP_API_BASE_URL is undefined.');
}

export interface SubArea {
    full_name: string;
    name: string;
    active: boolean;
}

export interface Area {
    id: string;
    name: string;
    next_action: string; // ISO string for datetime
    last_action: string; // ISO string for datetime
    last_action_success: boolean;
    sub_areas: SubArea[];
    number_name: string;
    last_error: string;
    flight_operating_hours: string[];
}

export interface ApiResponseArea {
    message: string;
    data: Area[];
}

interface Transcript {
    date: string;
    transcript: string;
}

export interface ApiResponseTranscript {
    message: string;
    data: Transcript;
}

export const fetchApiAreas = async (): Promise<ApiResponseArea> => {
    try {
        const response = await axios.get(`${API_BASE_URL}/api/v1/areas`);
        if (response.data) {
            return response.data;
        }
        console.error("Malformed data when fetching API:", response);
        throw new Error("Malformed data");

    // eslint-disable-next-line  @typescript-eslint/no-explicit-any
    } catch (error: any) {
        throw new Error(error.response?.data?.message || error.message || 'Unknown error occurred');
    }
};

export const fetchApiTranscript = async (area: string): Promise<ApiResponseTranscript> => {
    try {
        const response = await axios.get(`${API_BASE_URL}/api/v1/transcripts/${area}/latest`);
        if (response?.data) {
            return response.data;
        }
        console.error("Malformed data when fetching API:", response);
        throw new Error("Malformed data");

    // eslint-disable-next-line  @typescript-eslint/no-explicit-any
    } catch (error: any) {
        throw new Error(error.response?.data?.message || error.message || 'Unknown error occurred');
    }
};

interface FeatureStyling {
    Color: string;
    Opacity: number;
}

// Returns a matching SubArea for a given feature
const resolveSubAreaFromFeature = (feature: Feature<Geometry>, apiData: ApiResponseArea): SubArea | undefined => {
    const resolvedArea = resolveAreaFromFeature(feature, apiData);
    const matchingSubArea = resolvedArea?.sub_areas.find(subArea => {
        return subArea.full_name === feature?.properties?.Name;
    });
    if (matchingSubArea === undefined) {
        console.error("Could not resolve SubArea based on feature name:", feature?.properties?.Name);
    }

    return matchingSubArea;
};

  // Resolves a matching Area for a given feature
export const resolveAreaFromFeature = (feature: Feature<Geometry>, apiData: ApiResponseArea | null): Area => {
    const candidateName = feature?.properties?.Name.split(" ")[1].toLowerCase();
    const matchingArea = apiData?.data?.find(area => {
        return area.name === candidateName;
    });
    if (matchingArea === undefined || apiData === null) {
        console.error("Could not resolve Area based on candidate:", candidateName, "apiData is:", apiData);
        
        // Dummy Area
        return {
            id: "0",
            name: "Unknown",
            last_action: "",
            last_action_success: false,
            next_action: "",
            sub_areas: [{
                full_name: "Unknown",
                name: "Unknown",
                active: true,
            }],
            number_name: "",
            last_error: "",
            flight_operating_hours: [""],
        };
    }

    return matchingArea;
};

// Returns a color for a feature based on its corresponding SubAreas activeness
export const getStylingForFeature = (feature: Feature<Geometry> | undefined, apiData: ApiResponseArea): FeatureStyling => {
    const featureStyling: FeatureStyling = {
        Color: "yellow",
        Opacity: 1
    };

    if (!apiData || feature === undefined) {
        featureStyling.Color = "gray";
        return featureStyling;
    }

    const resolvedSubArea = resolveSubAreaFromFeature(feature, apiData);
    const resolvedArea = resolveAreaFromFeature(feature, apiData);

    if (resolvedArea?.last_action_success) {
        featureStyling.Color = resolvedSubArea?.active ? 'red' : 'green';
        featureStyling.Opacity = resolvedSubArea?.active ? 1 : 0.5;
        return featureStyling;
    }

    return featureStyling;
};