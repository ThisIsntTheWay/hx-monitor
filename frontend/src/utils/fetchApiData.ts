import axios from 'axios';

const API_BASE_URL = process.env.REACT_APP_API_BASE_URL;
if (!API_BASE_URL) {
    throw new Error('REACT_APP_API_BASE_URL is undefined.');
}

export interface SubArea {
    Fullname: string;
    Name: string;
    Status: boolean;
}

export interface Area {
    ID: string;
    Name: string;
    NextAction: string; // ISO string for datetime
    LastAction: string; // ISO string for datetime
    LastActionSuccess: boolean;
    SubAreas: SubArea[];
    NumberName: string;
    LastError: string;
    FlightOperatingHours: string[];
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
        return response.data;
    } catch (error: any) {
        throw new Error(error.response?.data?.message || error.message || 'Unknown error occurred');
    }
};

export const fetchApiTranscript = async (area: string): Promise<ApiResponseTranscript> => {
    try {
        const response = await axios.get(`${API_BASE_URL}/api/v1/transcripts/${area}/latest`);
        return response.data;
    } catch (error: any) {
        throw new Error(error.response?.data?.message || error.message || 'Unknown error occurred');
    }
};

interface FeatureStyling {
    Color: string;
    Opacity: number;
}

// Returns a matching SubArea for a given feature
const resolveSubAreaFromFeature = (feature: any, apiData: ApiResponseArea): SubArea | undefined => {
    const resolvedArea = resolveAreaFromFeature(feature, apiData);
    const matchingSubArea = resolvedArea?.SubAreas.find(subArea => {
        return subArea.Fullname === feature.properties.Name;
    });
    if (matchingSubArea === undefined) {
        console.error("Could not resolve SubArea based on feature name:", feature.properties.Name);
    }

    return matchingSubArea;
};

  // Resolves a matching Area for a given feature
export const resolveAreaFromFeature = (feature: any, apiData: ApiResponseArea | null): Area => {
    const candidateName = feature.properties.Name.split(" ")[1].toLowerCase();
    const matchingArea = apiData?.data?.find(area => {
        return area.Name === candidateName;
    });
    if (matchingArea === undefined || apiData === null) {
        console.error("Could not resolve Area based on candidate:", candidateName, "apiData is:", apiData);
        
        // Dummy Area
        return {
            ID: "0",
            Name: "Unknown",
            LastAction: "",
            LastActionSuccess: false,
            NextAction: "",
            SubAreas: [{
                Fullname: "Unknown",
                Name: "Unknown",
                Status: false,
            }],
            NumberName: "",
            LastError: "",
            FlightOperatingHours: [""],
        };
    }

    return matchingArea;
};

// Returns a color for a feature based on its correspinding SubAreas status
export const getStylingForFeature = (feature: any, apiData: ApiResponseArea): FeatureStyling => {
    const featureStyling: FeatureStyling = {
        Color: "yellow",
        Opacity: 1
    };

    if (!apiData) {
        featureStyling.Color = "gray";
        return featureStyling;
    }

    const resolvedSubArea = resolveSubAreaFromFeature(feature, apiData);
    const resolvedArea = resolveAreaFromFeature(feature, apiData);

    if (resolvedArea?.LastActionSuccess) {
        featureStyling.Color = resolvedSubArea?.Status ? 'red' : 'green';
        featureStyling.Opacity = resolvedSubArea?.Status ? 1 : 0.5;
        return featureStyling;
    }

    return featureStyling;
};