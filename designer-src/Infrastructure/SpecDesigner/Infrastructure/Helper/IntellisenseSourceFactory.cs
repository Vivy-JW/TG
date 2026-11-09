using System;
using System.Collections.Generic;
using System.Linq;
using System.Xml.Linq;
using SpecDesigner.Infrastructure.Model;

namespace SpecDesigner.Infrastructure.Helper
{
	// Token: 0x0200004D RID: 77
	public class IntellisenseSourceFactory
	{
		// Token: 0x06000252 RID: 594 RVA: 0x0000B250 File Offset: 0x00009450
		public static List<IIntellisenseModel> Parse(string source)
		{
			List<IIntellisenseModel> list = new List<IIntellisenseModel>();
			IEnumerable<XElement> enumerable = from query in XDocument.Parse(source).Descendants("com")
				select (query);
			foreach (XElement xelement in enumerable)
			{
				IntellisenseModel intellisenseModel = null;
				intellisenseModel = IntellisenseModel.Create((xelement.Attribute("id") != null) ? xelement.Attribute("id").Value : ((xelement.Attribute("name") != null) ? xelement.Attribute("name").Value : null));
				foreach (XAttribute xattribute in xelement.Attributes())
				{
					string text;
					if ((text = xattribute.Name.LocalName.ToLower()) != null && text == "desc" && intellisenseModel != null)
					{
						intellisenseModel.Description = xattribute.Value;
						intellisenseModel.Description = intellisenseModel.Description.Replace("\n", Environment.NewLine);
					}
				}
				foreach (XElement xelement2 in xelement.Elements())
				{
					ParameterModel parameterModel = new ParameterModel();
					if (xelement2.Name.LocalName.ToLower().StartsWith("param", StringComparison.CurrentCultureIgnoreCase))
					{
						parameterModel.Usage = UsageType.PARAMETER;
					}
					else if (xelement2.Name.LocalName.ToLower().StartsWith("rtn", StringComparison.CurrentCultureIgnoreCase))
					{
						parameterModel.Usage = UsageType.RETURN;
					}
					foreach (XAttribute xattribute2 in xelement2.Attributes())
					{
						string text2;
						if ((text2 = xattribute2.Name.LocalName.ToLower()) != null)
						{
							if (!(text2 == "name"))
							{
								if (!(text2 == "type"))
								{
									if (text2 == "desc")
									{
										parameterModel.Description = xattribute2.Value;
									}
								}
								else
								{
									parameterModel.Type = xattribute2.Value;
								}
							}
							else
							{
								parameterModel.Name = xattribute2.Value;
							}
						}
					}
					intellisenseModel.AppendParameter(parameterModel);
				}
				list.Add(intellisenseModel);
			}
			return list;
		}
	}
}
