#!/usr/bin/env python3
"""Chunked operations to bypass 20s sandbox limit"""
import json, os

def save_progress(progress_file, data):
    """Save progress to local file (survives between calls)"""
    with open(progress_file, 'w') as f:
        json.dump(data, f, indent=2)

def load_progress(progress_file):
    """Load progress from local file"""
    if not os.path.exists(progress_file):
        return {}
    with open(progress_file, 'r') as f:
        return json.load(f)

def chunked_process(items, process_func, progress_file, chunk_size=5):
    """Process items in chunks, save progress between chunks
    
    Args:
        items: list of items to process
        process_func: function(item, index) -> result
        progress_file: path to save progress
        chunk_size: items per chunk
    
    Usage:
        # Call 1: process first 5 items
        items = list(range(100))
        chunked_process(items, my_func, "progress.json", chunk_size=5)
        
        # Call 2: continue from where stopped
        progress = load_progress("progress.json")
        remaining = items[progress['completed']:]
        chunked_process(remaining, my_func, "progress.json", chunk_size=5)
    """
    progress = load_progress(progress_file)
    completed = progress.get('completed', 0)
    results = progress.get('results', [])
    
    for i, item in enumerate(items[:chunk_size]):
        index = completed + i
        try:
            result = process_func(item, index)
            results.append({"index": index, "status": "ok", "result": str(result)})
        except Exception as e:
            results.append({"index": index, "status": "error", "error": str(e)})
    
    completed += min(chunk_size, len(items))
    save_progress(progress_file, {"completed": completed, "results": results})
    
    print(f"Processed {completed}/{len(items) + completed} items")
    return completed
