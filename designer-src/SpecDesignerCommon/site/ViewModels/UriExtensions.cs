using System;
using System.Linq;

namespace SpecDesignerCommon.Site.ViewModels
{
	// Token: 0x02000083 RID: 131
	public static class UriExtensions
	{
		// Token: 0x06000539 RID: 1337 RVA: 0x0001818D File Offset: 0x0001638D
		public static Uri Append(this Uri uri, params string[] paths)
		{
			return new Uri(paths.Aggregate(uri.AbsoluteUri, (string current, string path) => string.Format("{0}/{1}", current.TrimEnd(new char[] { '/' }), path.TrimStart(new char[] { '/' }))));
		}
	}
}
