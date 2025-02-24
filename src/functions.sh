#!/bin/sh

format_duration() {
    local seconds=$1
    local days=$((seconds / 86400))
    local hours=$(((seconds % 86400) / 3600))
    local minutes=$(((seconds % 3600) / 60))
    local remaining_seconds=$((seconds % 60))
    
    local output=""
    [ $days -gt 0 ] && output="${output}${days}d "
    [ $hours -gt 0 ] && output="${output}${hours}h "
    [ $minutes -gt 0 ] && output="${output}${minutes}m "
    [ $remaining_seconds -gt 0 ] && output="${output}${remaining_seconds}s"
    
    echo "$output"
}

calculate_next_time() {
    # Get current timestamp
    now=$(date +%s)
    
    if [ "${CRON_SCHEDULE}" = "* * * * *" ]; then
        # For every minute schedule, next run is in 1 minute
        echo $((now + 60))
    else
        # Parse cron schedule to determine next run
        # This is a simplified version - you may want to add more cron pattern handling
        minute=$(echo "${CRON_SCHEDULE}" | awk '{print $1}')
        hour=$(echo "${CRON_SCHEDULE}" | awk '{print $2}')
        
        if [ "$minute" = "*" ] && [ "$hour" = "*" ]; then
            # Every minute
            echo $((now + 60))
        else
            # Default to next midnight if pattern isn't recognized
            today_midnight=$(date -d "00:00" +%s)
            tomorrow_midnight=$((today_midnight + 86400))
            if [ $now -lt $today_midnight ]; then
                echo $today_midnight
            else
                echo $tomorrow_midnight
            fi
        fi
    fi
}

format_schedule_message() {
    current_time=$(date +%s)
    target_time=$1
    remaining_seconds=$(( target_time - current_time ))
    human_readable=$(format_duration $remaining_seconds)
    
    echo "🕐 Current time: $(date '+%Y-%m-%d %H:%M:%S UTC')"
    echo "⏳ Next backup in ${human_readable} at $(date -d @$target_time '+%Y-%m-%d %H:%M:%S UTC')"
}