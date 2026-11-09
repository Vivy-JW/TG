using System;
using System.CodeDom.Compiler;
using System.ComponentModel;
using System.Diagnostics;
using System.Globalization;
using System.Resources;
using System.Runtime.CompilerServices;

namespace SpecDesigner.Search.Properties
{
	// Token: 0x0200000B RID: 11
	[CompilerGenerated]
	[GeneratedCode("System.Resources.Tools.StronglyTypedResourceBuilder", "4.0.0.0")]
	[DebuggerNonUserCode]
	internal class Resources
	{
		// Token: 0x06000090 RID: 144 RVA: 0x00003FAC File Offset: 0x000021AC
		internal Resources()
		{
		}

		// Token: 0x1700001E RID: 30
		// (get) Token: 0x06000091 RID: 145 RVA: 0x00003FB4 File Offset: 0x000021B4
		[EditorBrowsable(EditorBrowsableState.Advanced)]
		internal static ResourceManager ResourceManager
		{
			get
			{
				if (object.ReferenceEquals(Resources.resourceMan, null))
				{
					ResourceManager resourceManager = new ResourceManager("SpecDesigner.Search.Properties.Resources", typeof(Resources).Assembly);
					Resources.resourceMan = resourceManager;
				}
				return Resources.resourceMan;
			}
		}

		// Token: 0x1700001F RID: 31
		// (get) Token: 0x06000092 RID: 146 RVA: 0x00003FF3 File Offset: 0x000021F3
		// (set) Token: 0x06000093 RID: 147 RVA: 0x00003FFA File Offset: 0x000021FA
		[EditorBrowsable(EditorBrowsableState.Advanced)]
		internal static CultureInfo Culture
		{
			get
			{
				return Resources.resourceCulture;
			}
			set
			{
				Resources.resourceCulture = value;
			}
		}

		// Token: 0x04000033 RID: 51
		private static ResourceManager resourceMan;

		// Token: 0x04000034 RID: 52
		private static CultureInfo resourceCulture;
	}
}
