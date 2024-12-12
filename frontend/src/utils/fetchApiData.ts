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
}

export interface ApiResponse {
    message: string;
    data: Area[];
}

export const fetchApiData = async (endpoint: string): Promise<ApiResponse> => {
    try {
        const response = await axios.get(`${API_BASE_URL}${endpoint}`);
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
const resolveSubAreaFromFeature = (feature: any, apiData: ApiResponse): SubArea | undefined => {
    const resolvedArea = resolveAreaFromFeature(feature, apiData)
    const matchingSubArea = resolvedArea?.SubAreas.find(subArea => {
        return subArea.Fullname === feature.properties.Name;
    })
    if (matchingSubArea === undefined) {
        console.error("Could not resolve SubArea based on feature name:", feature.properties.Name)
    }

    return matchingSubArea;
}

  // Resolves a matching Area for a given feature
const resolveAreaFromFeature = (feature: any, apiData: ApiResponse): Area | undefined => {
    const candidateName = feature.properties.Name.split(" ")[1].toLowerCase();
    const matchingArea = apiData?.data?.find(area => {
        return area.Name === candidateName;
    });
    if (matchingArea === undefined) {
        console.error("Could not resolve Area based on candidate:", candidateName)
    }

    return matchingArea;
}

  // Returns a color for a feature based on its correspinding SubAreas status
  export const GetStylingForFeature = (feature: any, apiData: ApiResponse): FeatureStyling => {
    let featureStyling: FeatureStyling = {
        Color: "yellow",
        Opacity: 1
    }

    if (!apiData) {
        featureStyling.Color = "gray"
        return featureStyling
    }

    const resolvedSubArea = resolveSubAreaFromFeature(feature, apiData);
    const resolvedArea = resolveAreaFromFeature(feature, apiData);

    if (resolvedArea?.LastActionSuccess) {
        featureStyling.Color = resolvedSubArea?.Status ? 'red' : 'green';
        featureStyling.Opacity = resolvedSubArea?.Status ? 1 : 0.5;
        return featureStyling
    }

    return featureStyling
};